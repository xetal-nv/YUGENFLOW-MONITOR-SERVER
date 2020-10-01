package sensorManager

import (
	"fmt"
	"gateserver/codings"
	"gateserver/dataformats"
	"gateserver/gateManager"
	"gateserver/storage/sensorDB"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
	"xetal.ddns.net/utils/recovery"
)

/*
	initiate the TCP channels with all checks
	send data using gateManager to the proper gates
*/

func handler(conn net.Conn) {

	// support methods
	deadlineFailed := func(ipc string, e error) {
		if globals.DebugActive {
			fmt.Printf("sensorManager.handler: error on setting deadline for %v : %v\n", ipc, e)
		}
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + ipc,
				"error on setting deadline: " + e.Error(),
				[]int{0}, false})
	}
	failedRead := func(mach, ipc string, e error) {
		if globals.DebugActive {
			log.Printf("sensorManager.handler: error reading from %v//%v : %v\n", ipc, mach, e)
		}
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + mach,
				"read error",
				[]int{0}, true})
	}

	var sensorDef sensorDefinition
	enforceLoop := -1      // used to check the enforce tag execution
	mac := make([]byte, 6) // received amc address

	// cleaning up at closure
	defer func() {
		// We close the channel and update the sensor definition entry, when applicable
		_ = conn.Close()
		// if the mac is given, we need to reset all sensor data
		if sensorDef.mac != "" {
			// kill command process first
			if sensorDef.channels.reset != nil {
				select {
				case sensorDef.channels.reset <- true:
					<-sensorDef.channels.reset
				case <-time.After(time.Duration(globals.SensorTimeout)):
					// this might lead to a zombie that will kill itself eventually
				}

			}
			// remove entry from active sensor list
			ActiveSensors.Lock()
			delete(ActiveSensors.Id, sensorDef.id)
			delete(ActiveSensors.Mac, sensorDef.mac)
			ActiveSensors.Unlock()
			_ = sensorDB.DeleteDevice([]byte(sensorDef.mac))
			//startstopCommandProcess <- sensorDef.mac
			// release TCO token
			tokens <- nil
			if globals.DebugActive {
				fmt.Printf("sensorManager.tcpServer: released token, left: %v\n", len(tokens))
			}
		}
		//. in case of valid ID we remove the lookup entry as well
		if sensorDef.id != -1 {
			_ = sensorDB.DeleteLookUp([]byte{byte(sensorDef.id)})
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
				[]int{0}, true})
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
				[]int{0}, true})
		// A delay is inserted in case this is a malicious attempt and we mark the IP as suspicious
		others.WaitRandom(globals.MaliciousTimeout)
		_, _ = sensorDB.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
		return
	} else {
		mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", "", -1), " ")
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"device " + ipc,
				"last assigned MAC: " + mach,
				[]int{0}, true})
		// the mac is checked in the disabled list
		if banned, err := sensorDB.CheckMAC([]byte(mach), globals.MaliciousTriesMac); err == nil && banned {
			// We wait assuming it is an attack to slow it down
			others.WaitRandom(globals.MaliciousTimeout)
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensor " + mach,
					"malicious, connection refused",
					[]int{0}, true})
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
						[]int{0}, true})
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
			tcp:       conn,
			CmdAnswer: make(chan dataformats.Commandding, globals.ChannellingLength),
			Commands:  make(chan dataformats.Commandding, globals.ChannellingLength),
			reset:     make(chan bool, 1),
		}

		ActiveSensors.Mac[sensorDef.mac] = sensorDef.channels
		//startstopCommandProcess <- sensorDef.mac
		go recovery.RunWith(
			func() {
				sensorCommand(sensorDef.channels, sensorDef.mac)
			},
			nil)

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
						[]int{0}, true})
				return
			}
			if globals.DebugActive {
				fmt.Printf("refreshing EEPROM successful : %v\n", mach)
			}
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"EEPROM sensor " + mach,
					"refreshing successful",
					[]int{0}, true})
		}

		// sensor is marked as present but not active
		if err := sensorDB.MarkDeviceNotActive([]byte(mach)); err != nil {
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensor " + mach,
					"failed to add to device list",
					[]int{0}, true})
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
							[]int{0}, true})
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
						if globals.MalicioudMode > globals.OFF {
							sensorDef.failures += 1
							if sensorDef.failures > globals.FailureThreshold {
								_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
								// A delay is inserted in case this is a malicious attempt
								others.WaitRandom(globals.MaliciousTimeout)
								return
							}
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
										[]int{0}, true})
								//valid = false
								// with a wrong CRC the message is rejected but the connection is not closed
								if globals.CRCMaliciousCount {
									// in case of malicious mode severe we flag the mac and the IP
									if globals.MalicioudMode > globals.OFF {
										sensorDef.failures += 1
										if sensorDef.failures > globals.FailureThreshold {
											_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
											// A delay is inserted in case this is a malicious attempt
											others.WaitRandom(globals.MaliciousTimeout)
											return
										}
									}
								}
								continue
							}
						}

						// package is valid, we extract the ID
						sensorDef.idSent = int(data[1]) | int(data[0])<<8

						// if the sensor is not yet active, we go through the verifications needed to determine if the device is valid and active
						if !sensorDef.active {

							// check for possible attack on duplicated ID
							if sensorDef.id == -1 {
								ActiveSensors.RLock()
								if _, alreadyInUse := ActiveSensors.Id[sensorDef.idSent]; alreadyInUse {
									if globals.DebugActive {
										fmt.Printf("Potential malicious device using ID %v with mac %v and ip %v\n",
											sensorDef.idSent, mach, ipc)
									}
									// potential malicious attack, IP and MAc are marked
									ActiveSensors.RUnlock()
									_, _ = sensorDB.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
									_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
									// A delay is inserted in case this is a malicious attempt
									others.WaitRandom(globals.MaliciousTimeout)
									return
								}
								ActiveSensors.RUnlock()
							}
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
										[]int{0}, true})
								sensorDef.report = false
							}

							// if required we change the sensor ID itself
							if sensorDef.idSent != sensorDef.id && sensorDef.enforce {
								if err := setID(sensorDef.channels, sensorDef.id); err == nil {
									sensorDef.idSent = sensorDef.id
									sensorDef.enforce = false
									enforceLoop = enforceTries
								} else {
									if globals.DebugActive {
										fmt.Printf("Sensor %v has failed to change id\n", sensorDef.mac)
									}
									mlogger.Info(globals.SensorManagerLog,
										mlogger.LoggerData{"sensor " + mach,
											"has failed to change id",
											[]int{0}, true})
									// in case of malicious mode severe we flag the mac and the IP
									if globals.MalicioudMode > globals.OFF {
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
							sensorDef.enforce = false

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
												[]int{0}, true})
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
										[]int{0}, true})
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
												[]int{0}, true})
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

							if err := sensorDB.AddLookUp([]byte{byte(sensorDef.id)}, sensorDef.mac); err != nil {
								mlogger.Error(globals.SensorManagerLog,
									mlogger.LoggerData{"sensor " + mach,
										"failed to saving lookup declaration", []int{0}, true})
								return
							}

							if err := sensorDB.MarkDeviceActive([]byte(mach)); err == nil {
								sensorDef.active = true
								ActiveSensors.Lock()
								ActiveSensors.Id[sensorDef.id] = sensorDef.mac
								ActiveSensors.Unlock()
								mlogger.Info(globals.SensorManagerLog,
									mlogger.LoggerData{"sensor " + mach,
										"is active with ID " + strconv.Itoa(sensorDef.id),
										[]int{0}, true})
								gateManager.SensorList.RLock()
								//fmt.Println(sensorDef.id, gateManager.SensorList.DataChannel[sensorDef.id])
								//fmt.Println(sensorDef.id, gateManager.SensorList.GateList[sensorDef.id])
								sensorDef.channels.gateChannel = gateManager.SensorList.DataChannel[sensorDef.id]
								gateManager.SensorList.RUnlock()
							}
						}

						// the sensor can now be considered valid and we send the data to the gate
						if sensorDef.channels.gateChannel == nil {
							// somehow sensor definition got corrupted
							if globals.DebugActive {
								fmt.Printf("Sensor %v has no valid gate associated\n", sensorDef.mac)
							}
							mlogger.Info(globals.SensorManagerLog,
								mlogger.LoggerData{"sensor " + mach,
									"has no valid gate associated",
									[]int{0}, true})
							return
						} else {
							for _, ch := range sensorDef.channels.gateChannel {
								flow := int(data[2])
								if flow == 255 {
									flow = -1
								}
								//fmt.Println(sensorDef.id, "sending data", ch, flow)
								ch <- dataformats.FlowData{
									Type:    "sensor",
									Name:    mach,
									Id:      sensorDef.id,
									Ts:      time.Now().UnixNano(),
									Netflow: flow,
								}
							}
						}
						//gateManager.DistributeData(sensorDef.id, int(data[2]))

					}
				default:
					// we first check if this is a setID DoC attack
					if cmd[0] == cmdAPI["setid"].cmd && maliciousSetIdDOS(ipc, mach) {
						// this is a malicious device
						// A delay is inserted in case this is a malicious attempt
						others.WaitRandom(globals.MaliciousTimeout)
						return
					}
					if sensorDef.channels.CmdAnswer == nil {
						// process is corrupted, we must terminate it
						if globals.DebugActive {
							fmt.Printf("sensorManager.handler: sensor commands channel found invalid\n")
						}
						mlogger.Error(globals.SensorManagerLog,
							mlogger.LoggerData{"sensorManager.handler",
								"critical error, sensor commands channel found invalid",
								[]int{0}, false})
						return
					}
					// this is a command answer
					// only the answer to setid can be allowed when the sensor is not active and id!=idSent
					if sensorDef.active ||
						sensorDef.id != sensorDef.idSent && cmd[0] == cmdAPI["setid"].cmd {
						//(sensorDef.id == -1 && sensorDef.idSent == 65535) && cmd[0] == cmdAPI["setid"].cmd {
						// we verify that we received a command answer from an active device
						if v, ok := cmdAnswerLen[cmd[0]]; ok {
							// in case if no CRC the length needs to be decrease
							if !globals.CRCused {
								v -= 1
							}
							// check if command answer is fully correct and forward it to the command process
							if v == 0 {
								//this can only happen when CRC is not used
								select {
								case sensorDef.channels.CmdAnswer <- cmd:
									select {
									case ans := <-sensorDef.channels.CmdAnswer:
										if ans != nil {
											if globals.MalicioudMode > globals.OFF {
												sensorDef.failures += 1
												if sensorDef.failures > globals.FailureThreshold {
													_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
													// A delay is inserted in case this is a malicious attempt
													others.WaitRandom(globals.MaliciousTimeout)
													return
												}
											}
										}
									case <-time.After(time.Duration(globals.SensorTimeout*3) * time.Second):
										if globals.DebugActive {
											log.Printf("sensorManager.handler: hanging operation in receiving command answer\n")
										}
										return
									}

								case <-time.After(time.Duration(globals.SensorTimeout*3) * time.Second):
									if globals.DebugActive {
										log.Printf("sensorManager.handler: hanging operation in sending "+
											"command answer %v\n", cmd)
									}
									return
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
									if globals.MalicioudMode > globals.OFF {
										sensorDef.failures += 1
										if sensorDef.failures > globals.FailureThreshold {
											_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
											// A delay is inserted in case this is a malicious attempt
											others.WaitRandom(globals.MaliciousTimeout)
											return
										}
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
													[]int{0}, true})
											//valid = false
											// with a wrong CRC the message is rejected but the connection is not closed
											if globals.CRCMaliciousCount {
												// in case of malicious mode severe we flag the mac and the IP
												if globals.MalicioudMode > globals.OFF {
													sensorDef.failures += 1
													if sensorDef.failures > globals.FailureThreshold {
														_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
														// A delay is inserted in case this is a malicious attempt
														others.WaitRandom(globals.MaliciousTimeout)
														return
													}
												}
											}
											continue
										}
									}

									select {
									case sensorDef.channels.CmdAnswer <- cmd[:len(cmd)-1]:
										select {
										case ans := <-sensorDef.channels.CmdAnswer:
											if ans != nil {
												if globals.MalicioudMode > globals.OFF {
													sensorDef.failures += 1
													if sensorDef.failures > globals.FailureThreshold {
														_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
														// A delay is inserted in case this is a malicious attempt
														others.WaitRandom(globals.MaliciousTimeout)
														return
													}
												}
											}
											// in case we receive a valid answer to setid, we close the channel
											// this allows for the server to adapt to the new ID
											if cmd[0] == cmdAPI["setid"].cmd {
												_ = sensorDB.RemoveSuspectedMAC([]byte(sensorDef.mac))
												return
											}
										case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
											if globals.DebugActive {
												log.Printf("sensorManager.handler: hanging operation in receiving command answer\n")
											}
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
							_ = sensorDB.RemoveSuspectedMAC([]byte(sensorDef.mac))
						} else {
							// illegal command answer received
							if globals.MalicioudMode > globals.OFF {
								sensorDef.failures += 1
								if sensorDef.failures > globals.FailureThreshold {
									_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
									// A delay is inserted in case this is a malicious attempt
									others.WaitRandom(globals.MaliciousTimeout)
									return
								}
							}
						}
					} else {
						// command answer received from a non valid device
						if globals.DebugActive {
							fmt.Printf("sensorManager.handler: device %v//%v not active and sending non-data packages\n", ipc, mach)
						}
						if globals.MalicioudMode > globals.OFF {
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
			if enforceLoop >= 0 {
				if sensorDef.id != sensorDef.idSent {
					if enforceLoop == 0 {
						if globals.EnforceStrict {
							if globals.DebugActive {
								fmt.Printf("Sensor %v disconnected due to failed ID enforce\n", sensorDef.mac)
							}
							mlogger.Info(globals.SensorManagerLog,
								mlogger.LoggerData{"sensor " + mach,
									"enforce has failed, sensor disconnected",
									[]int{0}, true})
							return
						} else {
							if globals.DebugActive {
								fmt.Printf("Sensor %v failed ID enforce\n", sensorDef.mac)
							}
							mlogger.Info(globals.SensorManagerLog,
								mlogger.LoggerData{"sensor " + mach,
									"enforce has failed",
									[]int{0}, true})
							enforceLoop = -1
						}
					} else {
						enforceLoop--
						//println("not done")
					}
				} else {
					enforceLoop = -1
					//println("done")
				}
			}
		}

	}

}
