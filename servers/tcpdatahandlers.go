package servers

import (
	"fmt"
	"gateserver/codings"
	"gateserver/gates"
	"gateserver/support"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"time"
)

// handlers all TCP requests from a device
// it detects data, commands and command answers and act accordingly
// starts the associated handlerCommandAnswer

func handlerTCPRequest(conn net.Conn) {
	var deviceId int
	loop := true            // control variable
	pending := true         // control variable
	idKnown := false        // flags to know if the device has been recognised and initialised
	stop := make(chan bool) // channels used to reset the assocoated command thread
	mac := make([]byte, 6)  // received amc address
	var ci, ce chan []byte  // channel for the API command
	var devid chan int      // channel for the API command
	cks := errormngt[2]
	malfcheck := 0 // count number fo suspicions of attacks

	defer func() {
		if idKnown {
			// set TCP channel/ID couple flag to false
			mutexSensorMacs.Lock()
			sensorChanUsedID[deviceId] = false
			mutexSensorMacs.Unlock()
			// reset the command thread
			stop <- true
		}
		// clean up foe eventual unknown device
		mutexUnknownMac.Lock()
		delete(unknownMacChan, string(mac))
		mutexUnknownMac.Unlock()
		mutexUnusedDevices.Lock()
		delete(unusedDevice, deviceId)
		mutexUnusedDevices.Unlock()
		mutexPendingDevices.Lock()
		delete(pendingDevice, string(mac))
		mutexPendingDevices.Unlock()
		mutexConnMAC.Lock()
		delete(sensorConnMAC, string(mac))
		mutexConnMAC.Unlock()
		go func() {
			support.DLog <- support.DevData{"servers.handlerTCPRequest recover",
				support.Timestamp(), "", []int{1}, true}
		}()
		//noinspection GoUnhandledErrorResult
		conn.Close()
		tcpTokens <- true
		if support.Debug != 0 {
			log.Println("Releasing TCP token, remaining:", len(tcpTokens))
		}
	}()

	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]

	// Initially receive the MAC ip
	if _, e := conn.Read(mac); e != nil {
		mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", ":", -1), ":")
		log.Printf("servers.handlerTCPRequest: error on welcome message from %v//%v : %v\n", ipc, mach, e)
		// A delay is inserted in case this is a malicious attempt
		time.Sleep(time.Duration(timeout) * time.Second)
	} else {

		mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", ":", -1), ":")

		// reset all sensor parameters.
		// in case of errors the connection is closed/refused
		if e := setSensorParameters(conn, mach); e == nil {

			// Start reading data
			// define a malicious report function that, depending if on strict mode, also kills the connection
			//malf := func(strict bool) {
			malf := func() {
				if support.MalOn {
					//if strict {
					if malfcheck > 3 {
						log.Printf("servers.handlerTCPRequest: suspicious malicious device %v//%v\n", ipc, mach)
						go func() {
							support.DLog <- support.DevData{"servers.handlerTCPRequest: suspected malicious device " + mach + "@" + ipc,
								support.Timestamp(), "", []int{}, true}
						}()
						tsnow := support.Timestamp()
						for (tsnow + int64(maltimeout*1000)) > support.Timestamp() {
							if _, e := conn.Read(make([]byte, 256)); e != nil {
								break
							}
							time.Sleep(time.Duration(timeout) * (time.Second))
						}
						loop = false
					} else {
						log.Printf("servers.handlerTCPRequest: building suspicion on device %v//%v\n", ipc, mach)
					}
					//} else {
					//	log.Printf("servers.handlerTCPRequest: connected to an undeclared device %v//%v\n", ipc, mach)
					//}
				}
			}

			gates.MutexDeclaredDevices.RLock()
			if _, ok := gates.DeclaredDevices[string(mac)]; !ok {
				// Device is not allowed, behaviour depends if in strict mode
				if strictFlag {
					malfcheck += 1
					malf()
				} else {
					log.Printf("servers.handlerTCPRequest: connected to an undeclared device %v//%v\n", ipc, mach)
				}
				//malf(strictFlag)
			}
			gates.MutexDeclaredDevices.RUnlock()

			// START redo with different check

			mutexConnMAC.RLock()
			if oldcn, ts := sensorConnMAC[string(mac)]; ts {
				mutexConnMAC.RUnlock()
				// the current device is returning faster than its old channel is closed
				// or we are in presence of a malicious attack
				// checking if the old connection is alive will be used to decide the situation
				ch := make(chan bool)
				go func() {
					if _, e := oldcn.Read(make([]byte, 1)); e != nil {
						ch <- true
					} else {
						ch <- false
					}
				}()
				select {
				case ans := <-ch:
					if ans {
						mutexConnMAC.Lock()
						sensorConnMAC[string(mac)] = conn
						mutexConnMAC.Unlock()
						log.Printf("servers.handlerTCPRequest: re-connected to old device %v//%v\n", ipc, mach)
					} else {
						//malf(true)
						malfcheck += 1
						malf()
					}
				case <-time.After(time.Duration(timeout+1) * time.Second):
					// We wait further before assuming this is a malicious attack
					// this is a rare situation that should never happen
					ch0 := make(chan bool)
					go func() {
						if _, e := oldcn.Read(make([]byte, 1)); e != nil {
							ch0 <- true
						} else {
							ch0 <- false
						}
					}()
					select {
					case ans := <-ch0:
						if ans {
							log.Printf("servers.handlerTCPRequest: re-connected (timeout) to old device %v//%v\n", ipc, mach)
						} else {
							malfcheck += 1
							malf()
							//malf(true)
						}
					case <-time.After(time.Duration(maltimeout-timeout) * time.Second):
						malfcheck += 1
						malf()
						//malf(true)
					}
				}

			} else {
				mutexConnMAC.RUnlock()
				log.Printf("servers.handlerTCPRequest: connected to new device %v//%v\n", ipc, mach)
			}
			if loop {
				// in case the device is legit the connection table is updated
				mutexConnMAC.Lock()
				sensorConnMAC[string(mac)] = conn
				mutexConnMAC.Unlock()
			}

			if loop {

				// Update the list of pending devices
				mutexPendingDevices.Lock()
				pendingDevice[string(mac)] = true
				mutexPendingDevices.Unlock()

				// reset the sensor if it its first connection
				// malicious devices will also be receiving this command
				if support.RstON {
					mutexSensorMacs.RLock()
					mutexUnknownMac.RLock()
					_, ok1 := sensorIdMAC[string(mac)]
					_, ok2 := unkownDevice[string(mac)]
					mutexUnknownMac.RUnlock()
					mutexSensorMacs.RUnlock()
					if !(ok1 || ok2) {
						// this is a completely new device
						c := make(chan bool)
						// execute via goroutine to use timeout on the connection
						go func(c chan bool) {
							msg := []byte{cmdAPI["rstbg"].cmd}
							msg = append(msg, codings.Crc8(msg))
							if _, e = conn.Write(msg); e == nil {
								ans := make([]byte, 2)
								if _, e := conn.Read(ans); e != nil {
									// close connection in case of error
									if support.Debug != 0 {
										log.Printf("servers.handlerTCPRequest: failed start reset communication of device %v//%v\n", ipc, mach)
									}
									c <- false
								} else {
									//if ans[0] != cmdAPI["rstbg"].cmd || ans[1] != codings.Crc8([]byte{cmdAPI["rstbg"].cmd}) {
									if ans[0] != cmdAPI["rstbg"].cmd {
										if support.Debug != 0 {
											log.Printf("servers.handlerTCPRequest: failed start reset of device %v//%v with answer %v\n", ipc, mach, ans)
										}
										c <- false
									} else {
										if support.Debug != 0 {
											log.Printf("servers.handlerTCPRequest: executed start reset of device %v//%v\n", ipc, mach)
										}
										c <- true
									}
								}
							} else {
								// close connection in case of error
								if support.Debug != 0 {
									log.Printf("servers.handlerTCPRequest: connection error during start reset of device %v//%v\n", ipc, mach)
								}
								c <- false
							}
						}(c)
						select {
						case loop = <-c:
						case <-time.After(time.Duration(maltimeout) * time.Second):
							if support.Debug != 0 {
								log.Printf("servers.handlerTCPRequest: timeout on reset of device %v//%v\n", ipc, mach)
							}
							loop = false
						}
					}

				}

				if support.Debug != 0 {
					log.Printf("servers.handlerTCPRequest: status of device %v//%v is %v\n", ipc, mach, loop)
				}

				// updates SensorCmdMac if loop is still true checking if it already exists
				mutexSensorMacs.RLock()
				if loop {
					if chs, found := SensorCmdMac[mach]; !found {
						mutexSensorMacs.RUnlock()
						ci = make(chan []byte)
						ce = make(chan []byte)
						devid = make(chan int)
						mutexSensorMacs.Lock()
						SensorCmdMac[mach] = []chan []byte{ci}
						SensorCmdMac[mach] = append(SensorCmdMac[mach], ce)
						SensorIDCMDMac[mach] = devid
						mutexSensorMacs.Unlock()
						//go handlerCommandAnswer(conn, ci, ce, stop, devid)
						go handlerCommandAnswer(string(mac), ci, ce, stop, devid)
					} else {
						if devid, found = SensorIDCMDMac[mach]; !found {
							mutexSensorMacs.RUnlock()
							// system tables have been corrupted, a zombie is unavoidable
							log.Printf("servers.handlerTCPRequest: corrupted handlerCommandAnswer data for device %v//%v\n", ipc, mach)
							mutexSensorMacs.Lock()
							delete(SensorCmdMac, mach)
							mutexSensorMacs.Unlock()
							loop = false
						} else {
							mutexSensorMacs.RUnlock()
							ci = chs[0]
							ce = chs[1]
						}
					}
				} else {
					mutexSensorMacs.RUnlock()
				}
			}
			malfcheck = 0
			for loop {
				cmd := make([]byte, 1)
				if _, e := conn.Read(cmd); e != nil {

					if e == io.EOF {
						// in case of channel closed (EOF) it gets logged and the handler terminated
						log.Printf("servers.handlerTCPRequest: connection lost with device %v//%v\n", ipc, mach)
					} else {
						log.Printf("servers.handlerTCPRequest: error reading from %v//%v : %v\n", ipc, mach, e)
					}
					loop = false
				} else {
					switch cmd[0] {
					case 1:
						// Gate new counting data
						var data []byte
						if crcUsed {
							data = make([]byte, 4)
						} else {
							data = make([]byte, 3)
						}
						if _, e := conn.Read(data); e != nil {
							// A delay is inserted in case this is a malicious attempt
							time.Sleep(time.Duration(timeout) * time.Second)
							loop = false
							log.Printf("servers.handlerTCPRequest: error reading from %v//%v : %v\n", ipc, mach, e)
						} else {
							// Valid data
							valid := true
							if crcUsed {
								msg := append(cmd, data[:3]...)
								crc := codings.Crc8(msg)
								if crc != data[3] {
									if support.Debug > 0 {
										log.Print("servers.handlerTCPRequest: wrong CRC on received message\n")
									}
									valid = false
								}
							}
							if valid {
								// remove device from the pending list
								if pending {
									mutexPendingDevices.Lock()
									delete(pendingDevice, string(mac))
									mutexPendingDevices.Unlock()
									pending = false
								}
								// starts handlerCommandAnswer once with the proper ID
								// if the device was already connected the channels are already made and valid
								// first sample creates the command channels and handles if it does not exists
								if data[0] != 255 && data[1] != 255 {
									// Connected device has valid ID
									deviceId = int(data[1]) | int(data[0])<<8
									// declared devices need to be checked accounting for non strict mode
									// if mode is strict a request from a non registered device will never reach this point
									// a non registered device in non  strict mode should not be ignored
									ind := 65535
									gates.MutexDeclaredDevices.RLock()
									if v, ok := gates.DeclaredDevices[string(mac)]; ok {
										ind = v
									}
									gates.MutexDeclaredDevices.RUnlock()

									if ind != deviceId && ind != 65535 {
										// Device is a malicious attack, connection is terminated
										malfcheck += 2
										malf()
										//malf(true)
									} else {
										malfcheck = 0
										mutexSensorMacs.Lock()
										if !idKnown {
											oldMac, ok1 := sensorMacID[deviceId]
											oldId, ok2 := sensorIdMAC[string(mac)]
											_, ok3 := sensorChanID[deviceId]
											_, ok4 := SensorCmdID[deviceId]
											//  We check all entries as redundant check vs possible crashes or injection attacks
											if !(ok1 && ok2 && ok3 && ok4) {
												// this is a new device not previously connected
												sensorMacID[deviceId] = mac         // assign a mac to the id
												sensorIdMAC[string(mac)] = deviceId // assign an id to the mac
												sensorChanID[deviceId] = ci         // assign a channel to the id
												SensorCmdID[deviceId] = ce          // assign a command channel to the id
												sensorChanUsedID[deviceId] = true   // enable flag for TCP/Channel pair
												gates.SensorRst.Lock()
												go func(id int) { devid <- id }(deviceId)
												// enable periodic and self reset procedure
												if resetbg.valid {
													gates.SensorRst.Channel[deviceId] = make(chan bool, 0)
													go handlerReset(deviceId)
												}
												gates.SensorRst.Unlock()
											} else {
												// this is either a known device or an attack using a known/used ID
												if !reflect.DeepEqual(oldMac, mac) || (oldId != deviceId) {
													malfcheck += 2
													malf()
													//malf(true)
												} else {
													malfcheck = 0
													sensorChanUsedID[deviceId] = true
												}
											}
											idKnown = true
										}
										mutexSensorMacs.Unlock()
										if e := gates.SendData(deviceId, int(data[2])); e != nil {
											// when a not used (in the .env) device is found, it is placed in a list
											mutexUnusedDevices.Lock()
											if _, ok := unusedDevice[deviceId]; !ok {
												unusedDevice[deviceId] = string(mac)
												log.Println(e)
											}
											mutexUnusedDevices.Unlock()
										}
									}
								} else {
									mutexUnknownMac.Lock()
									// Connected device has invalid ID, needs to be set
									unkownDevice[string(mac)] = false
									if _, ok := unknownMacChan[string(mac)]; !ok {
										log.Printf("servers.handlerTCPRequest: new device with undefined id %v//%v\n", ipc, mach)
										unknownMacChan[string(mac)] = make(chan net.Conn)
									} else {
										log.Printf("servers.handlerTCPRequest: old device with undefined id %v//%v\n", ipc, mach)
									}
									s1 := make(chan bool, 1)
									s2 := make(chan bool, 1)
									go assingID(s1, conn, unknownMacChan[string(mac)], mac)
									mutexUnknownMac.Unlock()
									go func(terminate, stop chan bool) {
										loop := true
										for loop {
											select {
											case fl := <-terminate:
												stop <- fl
												loop = false
											case <-time.After(time.Duration(timeout) * time.Second):
												if _, e := conn.Read(make([]byte, 256)); e != nil {
													go func() { <-terminate }()
													stop <- false
													loop = false
												}
											}
										}
									}(s1, s2)
									loop = <-s2
								}
							}
						}
						cks = errormngt[2]
					default:
						if !idKnown {
							log.Printf("servers.handlerTCPRequest: rejected data %v from a device %v//%v with no valid ID yet\n", cmd[0], ipc, mach)
						} else {
							// verify it is a command answer, if not closes the TCP channel
							if v, ok := cmdAnswerLen[cmd[0]]; ok {
								if !crcUsed {
									v -= 1
								}
								if v == 0 {
									// this will never happen when CRC8 is used
									sensorChanID[deviceId] <- cmd
									// if the answer is incorrect the channel will be closed
									if ans := <-sensorChanID[deviceId]; ans != nil {
										if cks = cks - 1; cks <= 0 {
											loop = false
											log.Printf("servers.handlerTCPRequest: too many illegal commands %v sent by %v//%v\n", cmd[0], ipc, mach)
										}

									}
								} else {
									cmdd := make([]byte, v)
									if _, e := conn.Read(cmdd); e != nil {
										loop = false
										log.Printf("servers.handlerTCPRequest: error reading answer from %v//%v "+
											"for command %v\n", ipc, deviceId, cmd)
									} else {
										cmd = append(cmd, cmdd...)
										valid := true
										if crcUsed {
											crc := codings.Crc8(cmd[:len(cmd)-1])
											if crc != cmd[len(cmd)-1] {
												if support.Debug > 0 {
													log.Print("servers.handlerTCPRequest: wrong CRC on received message\n")
												}
												valid = false
											}
										}
										if valid {
											select {
											case sensorChanID[deviceId] <- cmd[:len(cmd)-1]:
												// in case we receive a valid answer to setid, we close the channel
												if cmd[0] == 14 {
													loop = false
												}
											case <-time.After(time.Duration(timeout) * time.Second):
												// internal issue, all goroutines will close on time out including the channel
												go func() { sensorChanID[deviceId] <- cmd }()
												log.Printf("servers.handlerTCPRequest: hanging operation in sending "+
													"command answer %v to user\n", cmd)
											}
										}
									}
								}
								cks = errormngt[2]
							} else {
								if cks = cks - 1; cks <= 0 {
									loop = false
									log.Printf("servers.handlerTCPRequest: too many illegal commands %v sent by %v//%v\n", cmd[0], ipc, mach)

								}
							}
						}
					}
				}
			}
			if idKnown {
				mutexSensorMacs.Lock()
				sensorChanUsedID[deviceId] = false
				mutexSensorMacs.Unlock()
			}
			log.Printf("servers.handlerTCPRequest: closing TCP channel to %v//%v\n", ipc, mach)
		} else {
			log.Printf("servers.handlerTCPRequest: closing TCP channel to %v//%v on specs setting for error %v\n", ipc, mach, e)
		}
	}
}
