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
	idKnown := false        // flags to know if the device has been recognised and initialised
	stop := make(chan bool) // channels used to reset the assocoated command thread
	mac := make([]byte, 6)  // received amc address

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
		mutexUnusedDevices.Lock()
		delete(unknownMacChan, string(mac))
		delete(unusedDevice, deviceId)
		mutexUnusedDevices.Unlock()
		mutexUnknownMac.Unlock()
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
		// Start reading data

		// define a malicious report function that, depending if on strict mode, also kills the connection
		malf := func(strict bool) {
			if strict {
				log.Printf("servers.handlerTCPRequest: suspicious malicious device %v//%v\n", ipc, mach)
				go func() {
					support.DLog <- support.DevData{"servers.handlerTCPRequest: suspected malicious device " + string(mac) + "@" + ipc,
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
				log.Printf("servers.handlerTCPRequest: connected to an undeclared device %v//%v\n", ipc, mach)
			}
		}

		gates.MutexDeclaredDevices.RLock()
		if _, ok := gates.DeclaredDevices[string(mac)]; !ok {
			// Device is not allowed, behaviour depends if in strict mode
			malf(strictFlag)
		}
		gates.MutexDeclaredDevices.RUnlock()
		mutexSensorMacs.Lock()
		if id, reged := sensorIdMAC[string(mac)]; reged {
			if active, ok := sensorChanUsedID[id]; ok {
				mutexSensorMacs.Unlock()
				if active {
					// We are in presence of a possible malicious attack
					// We wait maltimeout reading and throwing away periodically at timeout interval
					malf(true)
				} else {
					log.Printf("servers.handlerTCPRequest: connected to old device %v//%v\n", ipc, mach)
				}
			} else {
				// this should never happen
				// the following code resolves it in vase it happens for some unfrseen crash
				sensorChanUsedID[id] = false
				mutexSensorMacs.Unlock()
			}
		} else {
			mutexSensorMacs.Unlock()
			log.Printf("servers.handlerTCPRequest: connected to new device %v//%v\n", ipc, mach)
		}
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
							// starts handlerCommandAnswer once with the proper ID
							// if the device was already connected the channels are already made and valid
							// first sample creates the command channels and handles if it does not exists
							if data[0] != 255 && data[1] != 255 {
								// Connected device has valid ID
								deviceId = int(data[1]) | int(data[0])<<8
								// declared devicesd need to be checked accounting for non strict mode
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
									malf(true)
								} else {
									mutexSensorMacs.Lock()
									if !idKnown {
										oldMac, ok1 := sensorMacID[deviceId]
										oldId, ok2 := sensorIdMAC[string(mac)]
										_, ok3 := sensorChanID[deviceId]
										_, ok4 := SensorCmdID[deviceId]
										//  We check all entries as redundant check vs possible crashes or ainjection attacks
										if !(ok1 && ok2 && ok3 && ok4) {
											// this is a new device not previously connected
											sensorMacID[deviceId] = mac                // assign a mac to the id
											sensorIdMAC[string(mac)] = deviceId        // assign an id to the mac
											sensorChanID[deviceId] = make(chan []byte) // assign a channel to the id
											SensorCmdID[deviceId] = make(chan []byte)  // assign a command channel to the id
											sensorChanUsedID[deviceId] = true          // enable flag for TCP/Channel pair
											go handlerCommandAnswer(conn, sensorChanID[deviceId], SensorCmdID[deviceId], stop, deviceId)
											if resetbg.valid {
												go handlerReset(deviceId)
											}
										} else {
											// this is either a known device or an attack using a known/used ID
											if !reflect.DeepEqual(oldMac, mac) || (oldId != deviceId) {
												malf(true)
											} else {
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
								if _, ok := unknownMacChan[string(mac)]; !ok {
									log.Printf("servers.handlerTCPRequest: device with undefined id %v//%v\n", ipc, mach)
									s1 := make(chan bool, 1)
									s2 := make(chan bool, 1)
									unknownMacChan[string(mac)] = make(chan net.Conn, 1)
									unkownDevice[string(mac)] = false
									go assingID(s1, conn, unknownMacChan[string(mac)], mac)
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
									mutexUnknownMac.Unlock()
									loop = <-s2
								} else {
									mutexUnknownMac.Unlock()
								}
							}
						}
					}
				default:
					if !idKnown {
						loop = false
					} else {
						// verify it is a command answer, if not closes the TCP channel
						if v, ok := cmdAnswerLen[cmd[0]]; ok {
							if !crcUsed {
								v -= 1
							}
							if v == 0 {
								// this will never happen when CRC8 is used
								//fmt.Printf("Received something else %v\n", cmd)
								sensorChanID[deviceId] <- cmd
								// if the answer is incorrect the channel will be closed
								if ans := <-sensorChanID[deviceId]; ans != nil {
									loop = false
									log.Printf("servers.handlerTCPRequest: illegal operation %v sent by %v//%v\n", cmd[0], ipc, mach)
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
										case <-time.After(time.Duration(timeout) * time.Second):
											// internal issue, all goroutines will close on time out including the channel
											go func() { sensorChanID[deviceId] <- cmd }()
											log.Printf("servers.handlerTCPRequest: hanging operation in sending "+
												"command answer %v to user\n", cmd)
										}
									}
								}
							}
						} else {
							loop = false
							log.Printf("servers.handlerTCPRequest: illegal command %v sent by %v//%v\n", cmd[0], ipc, mach)
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

	}
}
