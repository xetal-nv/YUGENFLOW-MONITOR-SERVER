package sensorManager

import (
	"fmt"
	"gateserver/codings"
	"gateserver/dataformats"
	"gateserver/dbs/sensorDB"
	"gateserver/gateManager"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

/*
	initiate the TCP channels with all checks
	send data using gateManager to the proper gates
*/

// TODO command process needs to be done still !!!
func handler(conn net.Conn) {

	// support methods
	deadlineFailed := func(ipc string, e error) {
		if globals.DebugActive {
			fmt.Printf("sensorManager.handler: error on setting deadline for %v : %v\n", ipc, e)
		}
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + ipc,
				"error on setting deadline: " + e.Error(),
				[]int{}, false})
	}
	failedRead := func(mach, ipc string, e error) {
		if globals.DebugActive {
			log.Printf("sensorManager.handler: error reading from %v//%v : %v\n", ipc, mach, e)
		}
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + mach,
				"read error",
				[]int{}, true})
	}

	var sensorDef sensorDefinition
	mac := make([]byte, 6) // received amc address

	// cleaning up at closure
	defer func() {

		// We close the channel and update the sensor definition entry, when applicable
		_ = conn.Close()
		if sensorDef.mac != "" {
			// TODO kill command process first
			if sensorDef.channels.Reset != nil {
				sensorDef.channels.Reset <- true
				go func() { <-sensorDef.channels.Reset }()
			}
			ActiveSensors.Lock()
			delete(ActiveSensors.Id, sensorDef.id)
			delete(ActiveSensors.Mac, sensorDef.mac)
			ActiveSensors.Unlock()
			_ = sensorDB.DeleteDevice([]byte(sensorDef.mac))
			//startstopCommandProcess <- sensorDef.mac
		}

	}()

	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]
	// the IP is checked in the disabled list
	if banned, err := sensorDB.CheckIP([]byte(ipc), globals.MaliciousTriesIP); err == nil && banned {
		// We wait assuming it is an attack to slow it down
		others.WaitRandom(globals.MaliciousTimeout)
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"device " + ipc,
				"malicious, connection refused",
				[]int{}, true})
		return
	}

	// Initially receive the MAC value to identify the sensor
	if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
		deadlineFailed(ipc, e)
		return
	}
	if _, e := conn.Read(mac); e != nil {
		if globals.DebugActive {
			log.Printf("sensorManager.handler: error reading MAC from device with IP %v : %v\n", ipc, e)
		}
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"device " + ipc,
				"error reading MAC",
				[]int{}, true})
		// A delay is inserted in case this is a malicious attempt and we mark the IP as suspicious
		others.WaitRandom(globals.MaliciousTimeout)
		_, _ = sensorDB.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
		return
	} else {
		mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", "", -1), " ")
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"device " + ipc,
				"last assigned MAC: " + mach,
				[]int{}, true})
		// the mac is checked in the disabled list
		if banned, err := sensorDB.CheckMAC([]byte(mach), globals.MaliciousTriesMac); err == nil && banned {
			// We wait assuming it is an attack to slow it down
			others.WaitRandom(globals.MaliciousTimeout)
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensor " + mach,
					"malicious, connection refused",
					[]int{}, true})
			return
		}

		// read sensor definition form the sensor DB and store it locally
		def, erDB := sensorDB.ReadDefinition([]byte(mach))
		if erDB != nil {
			// the sensor has no definition
			def = dataformats.SensorDefinition{
				Id:      -1,
				Bypass:  false,
				Report:  false,
				Enforce: false,
				Strict:  false,
			}
		}
		sensorDef = sensorDefinition{
			mac:      mach,
			id:       def.Id,
			bypass:   def.Bypass,
			report:   def.Report,
			enforce:  def.Enforce,
			strict:   def.Strict,
			accept:   !def.Bypass && !def.Report && !def.Enforce && !def.Strict,
			active:   false,
			failures: 0,
		}

		// sensor configuration data is retrieved and it is verified that no channel is already open
		ActiveSensors.Lock()
		var alreadyInUse bool
		sensorDef.channels, alreadyInUse = ActiveSensors.Mac[mach]

		if alreadyInUse {
			ActiveSensors.Unlock()
			// The sensor has already an assigned TCP channel
			// We wait to see if it closes, if not the new connection channel is closed and marked as a possible attack
			time.Sleep(time.Duration(globals.SensorTimeout) * time.Second)
			ActiveSensors.Lock()
			sensorDef.channels, alreadyInUse = ActiveSensors.Mac[mach]
			if alreadyInUse {
				ActiveSensors.Unlock()
				sensorDef.mac = ""
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensor " + mach,
						"suspected malicious connection",
						[]int{}, true})
				// We wait assuming it is an attack to slow it down and mark the IP as suspicious
				others.WaitRandom(globals.MaliciousTimeout)
				_, _ = sensorDB.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
				return
			}
		}
		// the sensor is returned and this is not suspected to be a malicious attack
		// the IP is removed from the suspected list (if present)
		go func() {
			err := globals.SensorDBError
			for i := 0; i < 5 && err != nil; i++ {
				err = sensorDB.RemoveSuspecteIP([]byte(ipc))
			}
		}()
		sensorDef.channels = SensorChannel{
			Tcp:      conn,
			Commands: make(chan dataformats.CommandAnswer, globals.ChannellingLength),
			Reset:    make(chan bool, 1),
		}

		ActiveSensors.Mac[sensorDef.mac] = sensorDef.channels
		//startstopCommandProcess <- sensorDef.mac
		go sensorCommand(sensorDef.channels, sensorDef.mac)
		ActiveSensors.Unlock()

		// if enabled, the EEPROM is refreshed
		if globals.SensorEEPROMResetEnabled {
			if e := refreshEEPROM(conn, mach); e != nil {
				if globals.DebugActive {
					fmt.Printf("sensorManager.handler: closing TCP channel to %v//%v on "+
						"EEPROM refresh error : %v\n", ipc, mach, e)
				}
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensor " + mach,
						"error refreshing EEPROM" + e.Error(),
						[]int{}, true})
				return
			}
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensor " + mach,
					"refreshing EEPROM successful",
					[]int{}, true})
		}

		// sensor is marked as present but not active
		if err := sensorDB.MarkDeviceNotActive([]byte(mach)); err != nil {
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensor " + mach,
					"failed to add to device list",
					[]int{}, true})
			return
		}

		if e := conn.SetDeadline(time.Time{}); e != nil {
			deadlineFailed(ipc, e)
		}

		for {
			cmd := make([]byte, 1)
			if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
				deadlineFailed(mach, e)
				return
			}
			if _, e := conn.Read(cmd); e != nil {
				if e == io.EOF {
					// in case of channel closed (EOF) it gets logged and the handler terminated
					if globals.DebugActive {
						fmt.Printf("sensorManager.handler: connection lost with device %v//%v\n", ipc, mach)
					}
					mlogger.Info(globals.SensorManagerLog,
						mlogger.LoggerData{"sensor " + mach,
							"connection lost",
							[]int{}, true})
				} else {
					failedRead(mach, ipc, e)
				}
				return
			} else {
				switch cmd[0] {
				case 1:
					// this is a data packet
					var data []byte
					if globals.CRCused {
						data = make([]byte, 4)
					} else {
						data = make([]byte, 3)
					}
					if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
						deadlineFailed(mach, e)
						return
					}
					if _, e := conn.Read(data); e != nil {
						failedRead(mach, ipc, e)
						// in case of malicious mode severe we flag the mac and the IP
						sensorDef.failures += 1
						if sensorDef.failures > globals.FailureThreshold {
							_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
							// A delay is inserted in case this is a malicious attempt
							others.WaitRandom(globals.MaliciousTimeout)
							return
						}
					} else {
						// potentially valid package
						if globals.CRCused {
							msg := append(cmd, data[:3]...)
							crc := codings.Crc8(msg)
							if crc != data[3] {
								if globals.DebugActive {
									fmt.Print("servers.handlerTCPRequest: wrong CRC on received message\n")
								}
								mlogger.Info(globals.SensorManagerLog,
									mlogger.LoggerData{"sensor " + mach,
										"wrong CRC on received message",
										[]int{}, true})
								//valid = false
								// with a wrong CRC the message is rejected but the connection is not closed
								if globals.CRCMaliciousCount {
									// in case of malicious mode severe we flag the mac and the IP
									sensorDef.failures += 1
									if sensorDef.failures > globals.FailureThreshold {
										_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
										// A delay is inserted in case this is a malicious attempt
										others.WaitRandom(globals.MaliciousTimeout)
										return
									}
								}
								continue
							}
						}

						// package is valid, we extract the ID
						sensorDef.idSent = int(data[1]) | int(data[0])<<8

						// if the sensor is not yet active, we go through the verifications needed to determine if the device is valid and active
						if !sensorDef.active {
							if globals.DebugActive {
								fmt.Printf("New Sensor with Definition: %+v\n", sensorDef)
							}

							// we report an ID mismatch (the first connection only)
							if sensorDef.idSent != sensorDef.id && sensorDef.report {
								// ID mismatch gets reported
								if globals.DebugActive {
									fmt.Printf("Sensor %v has ID mismatch\n", sensorDef.mac)
								}
								mlogger.Info(globals.SensorManagerLog,
									mlogger.LoggerData{"sensor " + mach,
										"has sent wrong id: " + strconv.Itoa(sensorDef.idSent),
										[]int{}, true})
								sensorDef.report = false
							}

							// if required we change the sensor ID itself
							if sensorDef.idSent != sensorDef.id && sensorDef.enforce {
								if err := setID(conn, sensorDef.id); err == nil {
									sensorDef.idSent = sensorDef.id
									sensorDef.enforce = false
								} else {
									if globals.DebugActive {
										fmt.Printf("Sensor %v has failed to change id\n", sensorDef.mac)
									}
									mlogger.Info(globals.SensorManagerLog,
										mlogger.LoggerData{"sensor " + mach,
											"has failed to change id",
											[]int{}, true})
									// in case of malicious mode severe we flag the mac and the IP
									sensorDef.failures += 1
									if sensorDef.failures > globals.FailureThreshold {
										_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
										// A delay is inserted in case this is a malicious attempt
										others.WaitRandom(globals.MaliciousTimeout)
										return
									}
									continue
								}
							}

							// in case the sent ID and the defined ID is -1 is is set to 0xffff,
							// the sensor is considered not active
							if sensorDef.id == -1 && sensorDef.idSent == 65535 {
								if reject, newDevice, err := sensorDB.MarkInvalidDevice([]byte(mach), globals.MaximumInvalidIDInternal); err == nil {
									if newDevice {
										// new device, we wait in case the id gets set
										if globals.DebugActive {
											fmt.Printf("Sensor %v has no valid id\n", sensorDef.mac)
										}
										mlogger.Info(globals.SensorManagerLog,
											mlogger.LoggerData{"sensor " + mach,
												"has no valid id yet",
												[]int{}, true})
										// a delay is added to reduce activity on sensors with invalid ID's
										others.WaitRandom(globals.MaliciousTimeout)
									} else if reject {
										// the device has been unset for too long. We mark its MAC and disconnect
										if err := sensorDB.RemoveInvalidDevice([]byte(mach)); err == nil {
											_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
											// A delay is inserted in case this is a malicious attempt
											others.WaitRandom(globals.MaliciousTimeout)
											return
										}
									}
								}
								continue
							}

							if err := sensorDB.RemoveInvalidDevice([]byte(mach)); err != nil {
								return
							}

							// the strict condition on the ID can now be checked
							if sensorDef.idSent != sensorDef.id && sensorDef.strict {
								// sensor mismatch is considered illegal
								if globals.DebugActive {
									fmt.Printf("Sensor %v has been rejected\n", sensorDef.mac)
								}
								mlogger.Info(globals.SensorManagerLog,
									mlogger.LoggerData{"sensor " + mach,
										"rejected due to wrong id: " + strconv.Itoa(sensorDef.idSent),
										[]int{}, true})
								_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
								// A delay is inserted in case this is a malicious attempt
								others.WaitRandom(globals.MaliciousTimeout)
								return
							}

							// we set the id using the device one in case of missing definition
							if sensorDef.id == -1 {
								sensorDef.id = sensorDef.idSent
							}

							// The sensor has a valid id, we need to check if it is being used
							// This will be useful with dynamic configurations
							if !gateManager.SensorUsed(sensorDef.id) {
								// the sensor is currently not used
								// we treat it as a sensor with an invalid id
								if reject, newDevice, err := sensorDB.MarkInvalidDevice([]byte(mach),
									globals.MaximumInvalidIDInternal); err == nil {
									if newDevice {
										// new device, we wait in case the id gets set
										if globals.DebugActive {
											fmt.Printf("Sensor %v:%v is not being used\n", sensorDef.mac, sensorDef.id)
										}
										mlogger.Info(globals.SensorManagerLog,
											mlogger.LoggerData{"sensor " + mach,
												"is not being used, ID: " + strconv.Itoa(sensorDef.id),
												[]int{}, true})
										// a delay is added to reduce activity on sensors with invalid ID's
										others.WaitRandom(globals.MaliciousTimeout)
									} else if reject {
										// the device has been unset for too long. We mark its MAC and disconnect
										if err := sensorDB.RemoveInvalidDevice([]byte(mach)); err == nil {
											_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
											// A delay is inserted in case this is a malicious attempt
											others.WaitRandom(globals.MaliciousTimeout)
											return
										}
									}
								}
								continue
							}

							go func() {
								err := globals.SensorDBError
								for i := 0; i < 5 && err != nil; i++ {
									err = sensorDB.RemoveInvalidDevice([]byte(mach))
								}
							}()

							go func() {
								err := globals.SensorDBError
								for i := 0; i < 5 && err != nil; i++ {
									err = sensorDB.RemoveSuspectedMAC([]byte(mach))
								}
							}()

							if err := sensorDB.MarkDeviceActive([]byte(mach)); err == nil {
								sensorDef.active = true
								mlogger.Info(globals.SensorManagerLog,
									mlogger.LoggerData{"sensor " + mach,
										"is active",
										[]int{}, true})
							}
						}

						// the sensor can now be considered valid and we send the data to the gate
						gateManager.DistributeData(sensorDef.id, int(data[2]))

					}
				default:
					// TODO de malicious must be reset here also !!!

					if sensorDef.channels.Commands == nil {
						// process is corrupted, we must terminate it
						if globals.DebugActive {
							fmt.Printf("sensorManager.handler: sensor commands channel found invalid\n")
						}
						mlogger.Error(globals.SensorManagerLog,
							mlogger.LoggerData{"sensorManager.handler",
								"critical error, sensor commands channel found invalid",
								[]int{}, false})
						return
					}
					// this is a command answer
					// only the answer to setid can be allowed when the sensor is not active
					if !sensorDef.active &&
						!(sensorDef.id == -1 && sensorDef.idSent == 65535 && cmd[0] == cmdAPI["setid"].cmd) {
						// command answer received from a non active sensor
						if globals.DebugActive {
							fmt.Printf("sensorManager.handler: device %v//%v not active and sending non-data packages\n", ipc, mach)
						}
						sensorDef.failures += 1
						if sensorDef.failures > globals.FailureThreshold {
							_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
							// A delay is inserted in case this is a malicious attempt
							others.WaitRandom(globals.MaliciousTimeout)
							return
						}
					} else {
						// we verify that we received a command answer from an active device
						if v, ok := cmdAnswerLen[cmd[0]]; ok {
							// in case if no CRC the length needs to be decrease
							if !globals.CRCused {
								v -= 1
							}
							// check if command answer is fully correct and forward it to the command process
							if v == 0 {
								//this can only happen when CRC is not used
								sensorDef.channels.Commands <- cmd
								if ans := <-sensorDef.channels.Commands; ans != nil {
									sensorDef.failures += 1
									if sensorDef.failures > globals.FailureThreshold {
										_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
										// A delay is inserted in case this is a malicious attempt
										others.WaitRandom(globals.MaliciousTimeout)
										return
									}
								}
							} else {
								cmdd := make([]byte, v)
								if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
									deadlineFailed(mach, e)
									return
								}
								if _, e := conn.Read(cmdd); e != nil {
									failedRead(mach, ipc, e)
									// in case of malicious mode severe we flag the mac and the IP
									sensorDef.failures += 1
									if sensorDef.failures > globals.FailureThreshold {
										_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
										// A delay is inserted in case this is a malicious attempt
										others.WaitRandom(globals.MaliciousTimeout)
										return
									}
								} else {
									cmd = append(cmd, cmdd...)
									//valid := true
									if globals.CRCused {
										crc := codings.Crc8(cmd[:len(cmd)-1])
										if crc != cmd[len(cmd)-1] {
											if globals.DebugActive {
												fmt.Print("sensorManager.handler: wrong CRC on received command answer\n")
											}
											mlogger.Info(globals.SensorManagerLog,
												mlogger.LoggerData{"sensor " + mach,
													"wrong CRC on received command answer",
													[]int{}, true})
											//valid = false
											// with a wrong CRC the message is rejected but the connection is not closed
											if globals.CRCMaliciousCount {
												// in case of malicious mode severe we flag the mac and the IP
												sensorDef.failures += 1
												if sensorDef.failures > globals.FailureThreshold {
													_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
													// A delay is inserted in case this is a malicious attempt
													others.WaitRandom(globals.MaliciousTimeout)
													return
												}
											}
											continue
										}
									}

									select {
									case sensorDef.channels.Commands <- cmd[:len(cmd)-1]:
										// in case we receive a valid answer to setid, we close the channel
										if cmd[0] == cmdAPI["setid"].cmd {
											return
										}
									case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
										// internal issue, all goroutines will close on time out including the channel
										if globals.DebugActive {
											log.Printf("sensorManager.handler: hanging operation in sending "+
												"command answer %v\n", cmd)
										}
									}
								}
							}
						} else {
							// illegal command answer received
							sensorDef.failures += 1
							if sensorDef.failures > globals.FailureThreshold {
								_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
								// A delay is inserted in case this is a malicious attempt
								others.WaitRandom(globals.MaliciousTimeout)
								return
							}
						}
					}
				}
			}
		}

	}

}
