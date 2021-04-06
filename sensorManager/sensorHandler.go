package sensorManager

import (
	"fmt"
	"gateserver/codings"
	"gateserver/dataformats"
	"gateserver/gateManager"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"io"
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

func sensorHandler(conn net.Conn) {

	// support methods
	deadlineFailed := func(ipc string, e error) {
		if globals.DebugActive {
			fmt.Printf("sensorManager.sensorHandler: error on setting deadline for %v : %v\n", ipc, e)
		}
		mlogger.Error(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + ipc,
				"setting deadline: " + e.Error(),
				[]int{0}, false})
	}
	failedRead := func(mach, ipc string, e error) {
		if globals.DebugActive {
			fmt.Printf("sensorManager.sensorHandler: error reading from %v//%v : %v\n", ipc, mach, e)
		}
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + mach,
				"read error",
				[]int{0}, true})
	}

	if globals.EchoMode {
		defer func() {
			_ = conn.Close()
			//println("ok")
			// this is will not stay a zombie
			tokens <- nil
			if globals.DebugActive {
				fmt.Printf("sensorManager.tcpServer: released token, left: %v\n", len(tokens))
			}
		}()
		mac := make([]byte, 6)
		ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]
		if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
			fmt.Printf("sensorManager.sensorHandler: error on setting deadline for %v : %v\n", ipc, e)
			return
		}
		if _, e := conn.Read(mac); e != nil {
			fmt.Printf("sensorManager.sensorHandler: error reading MAC from device with IP %v : %v\n", ipc, e)
			return
		} else {
			mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", ":", -1), " ")
			mach = strings.Trim(mach, ":")
			fmt.Printf("Device with mac %v connected\n", mach)
			for {
				cmd := make([]byte, 1)
				if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
					fmt.Printf("sensorManager.sensorHandler: error on setting deadline for %v : %v\n", ipc, e)
					return
				}
				if _, e := conn.Read(cmd); e != nil {
					fmt.Printf("sensorManager.sensorHandler: error reading from device %v : %v\n", ipc, mach)
					return
				}
				var data []byte
				var ll int
				switch cmd[0] {
				case 1:
					// data
					ll = 4
				default:
					// command
					ll = CmdAnswerLen[cmd[0]]
				}
				if !globals.CRCused {
					ll -= 1
				}
				data = make([]byte, ll)
				if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
					deadlineFailed(mach, e)
					return
				}
				if _, e := conn.Read(data); e != nil {
					fmt.Printf("sensorManager.sensorHandler: error reading from device %v : %v\n", ipc, mach)
					return
				} else {
					cmd = append(cmd, data...)
					fmt.Printf("Trace device %v -> % x\n", mach, cmd)
				}
			}
		}
	} else {
		var sensorDef sensorDefinition
		enforceLoop := -1      // used to check the enforce tag execution
		mac := make([]byte, 6) // received amc address

		// cleaning up at closure
		defer func() {
			// We close the channel and update the sensor definition entry, when applicable
			// if the mac is given, we need to reset all sensor data
			// kill command process first
			if sensorDef.channels.reset != nil {
				select {
				case sensorDef.channels.reset <- true:
					go func() {
						select {
						case <-sensorDef.channels.reset:
						case <-time.After(time.Duration(globals.SensorTimeout)):
							// this might lead to a zombie that will kill itself eventually
						}
					}()
				case <-time.After(time.Duration(globals.SensorTimeout)):
					// this might lead to a zombie that will kill itself eventually
				}

			}
			if sensorDef.mac != "" {
				// remove entry from active sensor list
				ActiveSensors.Lock()
				delete(ActiveSensors.Id, sensorDef.id)
				delete(ActiveSensors.Mac, sensorDef.mac)
				ActiveSensors.Unlock()
				_ = diskCache.DeleteDevice([]byte(sensorDef.mac))
				// release TCO token
			}
			// in case of valid ID we remove the lookup entry as well
			if sensorDef.id != -1 {
				_ = diskCache.DeleteLookUp([]byte{byte(sensorDef.id)})
			}
			_ = conn.Close()
			// there will always be a received, no need to do a timeout
			tokens <- nil
			if globals.DebugActive {
				fmt.Printf("sensorManager.tcpServer: released token, left: %v\n", len(tokens))
			}
		}()

		ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]
		// the IP is checked in the disabled list
		if banned, err := diskCache.CheckIP([]byte(ipc), globals.MaliciousTriesIP); err == nil && banned {
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
				fmt.Printf("sensorManager.sensorHandler: error reading MAC from device with IP %v : %v\n", ipc, e)
			}
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"device " + ipc,
					"error reading MAC",
					[]int{0}, true})
			// A delay is inserted in case this is a malicious attempt and we mark the IP as suspicious
			others.WaitRandom(globals.MaliciousTimeout)
			_, _ = diskCache.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
			return
		} else {
			macStringified := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", "", -1), " ")
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"device " + ipc,
					"last assigned MAC: " + macStringified,
					[]int{0}, true})
			// the mac is checked in the disabled list
			if banned, err := diskCache.CheckMAC([]byte(macStringified), globals.MaliciousTriesMac); err == nil && banned {
				// We wait assuming it is an attack to slow it down
				others.WaitRandom(globals.MaliciousTimeout)
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensor " + macStringified,
						"malicious, connection refused",
						[]int{0}, true})
				return
			}

			// read sensor definition form the sensor DB and store it locally
			def, erDB := diskCache.ReadDefinition([]byte(macStringified))
			if erDB != nil {
				// the sensor has no definition
				def = dataformats.SensorDefinition{
					Id:      -1,
					MaxRate: def.MaxRate,
					Bypass:  false,
					Report:  false,
					Enforce: false,
					Strict:  false,
				}
			}
			sensorDef = sensorDefinition{
				mac:      macStringified,
				id:       def.Id,
				maxRate:  def.MaxRate,
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
			sensorDef.channels, alreadyInUse = ActiveSensors.Mac[macStringified]

			if alreadyInUse {
				ActiveSensors.Unlock()
				// The sensor has already an assigned TCP channel
				// We wait to see if it closes, if not the new connection channel is closed and marked as a possible attack
				time.Sleep(time.Duration(globals.SensorTimeout) * time.Second)
				ActiveSensors.Lock()
				sensorDef.channels, alreadyInUse = ActiveSensors.Mac[macStringified]
				if alreadyInUse {
					ActiveSensors.Unlock()
					sensorDef.mac = ""
					mlogger.Info(globals.SensorManagerLog,
						mlogger.LoggerData{"sensor " + macStringified,
							"suspected malicious connection",
							[]int{0}, true})
					// We wait assuming it is an attack to slow it down and mark the IP as suspicious
					others.WaitRandom(globals.MaliciousTimeout)
					_, _ = diskCache.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
					return
				}
			}
			// the sensor is returned and this is not suspected to be a malicious attack
			// the IP is removed from the suspected list (if present)
			go func() {
				err := globals.SensorDBError
				for i := 0; i < 5 && err != nil; i++ {
					err = diskCache.RemoveSuspectedIP([]byte(ipc))
				}
			}()
			sensorDef.channels = SensorChannel{
				Tcp:       conn,
				CmdAnswer: make(chan dataformats.Commanding, globals.ChannellingLength),
				Commands:  make(chan dataformats.Commanding, globals.ChannellingLength),
				reset:     make(chan bool, 1),
			}

			ActiveSensors.Mac[sensorDef.mac] = sensorDef.channels
			go recovery.RunWith(
				func() {
					sensorCommand(sensorDef.channels, sensorDef.mac)
				},
				nil)

			ActiveSensors.Unlock()

			// if enabled, the EEPROM is refreshed
			if globals.SensorEEPROMResetEnabled {
				if e := refreshEEPROM(conn, macStringified); e != nil {
					if globals.DebugActive {
						fmt.Printf("sensorManager.sensorHandler: closing TCP channel to %v//%v on "+
							"EEPROM refresh error : %v\n", ipc, macStringified, e)
					}
					mlogger.Info(globals.SensorManagerLog,
						mlogger.LoggerData{"sensor " + macStringified,
							"error refreshing EEPROM" + e.Error(),
							[]int{0}, true})
					return
				}
				if globals.DebugActive {
					fmt.Printf("refreshing EEPROM successful : %v\n", macStringified)
				}
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"EEPROM sensor " + macStringified,
						"refreshing successful",
						[]int{0}, true})
			}

			// sensor is marked as present but not active
			if err := diskCache.MarkDeviceNotActive([]byte(macStringified)); err != nil {
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensor " + macStringified,
						"failed to add to device list",
						[]int{0}, true})
				return
			}

			if e := conn.SetDeadline(time.Time{}); e != nil {
				deadlineFailed(ipc, e)
			}

			var lastSampleTS int64 = 0

			for {
				cmd := make([]byte, 1)
				if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
					deadlineFailed(macStringified, e)
					return
				}
				if _, e := conn.Read(cmd); e != nil {
					if e == io.EOF {
						// in case of channel closed (EOF) it gets logged and the sensorHandler terminated
						if globals.DebugActive {
							fmt.Printf("sensorManager.sensorHandler: connection lost with device %v//%v\n", ipc, macStringified)
						}
						mlogger.Info(globals.SensorManagerLog,
							mlogger.LoggerData{"sensor " + macStringified,
								"connection lost",
								[]int{0}, true})
					} else {
						failedRead(macStringified, ipc, e)
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
							deadlineFailed(macStringified, e)
							return
						}
						if _, e := conn.Read(data); e != nil {
							failedRead(macStringified, ipc, e)
							// in case of malicious mode severe we flag the mac and the IP
							if globals.MalicioudMode > globals.OFF {
								sensorDef.failures += 1
								if sensorDef.failures > globals.FailureThreshold {
									_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
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
										mlogger.LoggerData{"sensor " + macStringified,
											"wrong CRC on received message",
											[]int{0}, true})
									// with a wrong CRC the message is rejected but the connection is not closed
									if globals.CRCMaliciousCount {
										// in case of malicious mode severe we flag the mac and the IP
										if globals.MalicioudMode > globals.OFF {
											sensorDef.failures += 1
											if sensorDef.failures > globals.FailureThreshold {
												_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
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
								// which ID is used depends on the possible sensor definition
								var idTobeChecked int
								if sensorDef.id == -1 {
									idTobeChecked = sensorDef.idSent
								} else {
									idTobeChecked = sensorDef.id
								}
								ActiveSensors.RLock()
								if _, alreadyInUse := ActiveSensors.Id[idTobeChecked]; alreadyInUse {
									if globals.DebugActive {
										fmt.Printf("Potential malicious device using ID %v with mac %v and ip %v\n",
											sensorDef.idSent, macStringified, ipc)
									}
									// potential malicious attack, IP and MAc are marked
									ActiveSensors.RUnlock()
									_, _ = diskCache.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
									_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
									// A delay is inserted in case this is a malicious attempt
									others.WaitRandom(globals.MaliciousTimeout)
									return
								}
								ActiveSensors.RUnlock()
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
										mlogger.LoggerData{"sensor " + macStringified,
											"has sent wrong id: " + strconv.Itoa(sensorDef.idSent),
											[]int{0}, true})
									sensorDef.report = false
								}

								// if required we change the sensor ID itself
								if sensorDef.idSent != sensorDef.id && sensorDef.enforce {
									if err := setID(sensorDef.channels, sensorDef.id); err == nil {
										sensorDef.idSent = sensorDef.id
										sensorDef.enforce = false
									} else if enforceLoop == -1 {
										enforceLoop = enforceTries
										if globals.DebugActive {
											fmt.Printf("Sensor %v has failed to change id\n", sensorDef.mac)
										}
										mlogger.Info(globals.SensorManagerLog,
											mlogger.LoggerData{"sensor " + macStringified,
												"has failed to change id",
												[]int{0}, true})
										// in case of malicious mode severe we flag the mac and the IP
										if globals.MalicioudMode > globals.OFF {
											sensorDef.failures += 1
											if sensorDef.failures > globals.FailureThreshold {
												_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
												// A delay is inserted in case this is a malicious attempt
												others.WaitRandom(globals.MaliciousTimeout)
												return
											}
										}
										continue
									} else if enforceLoop -= 1; enforceLoop == 0 {
										if globals.EnforceStrict {
											if globals.DebugActive {
												fmt.Printf("Sensor %v disconnected due to failed ID enforce\n", sensorDef.mac)
											}
											mlogger.Info(globals.SensorManagerLog,
												mlogger.LoggerData{"sensor " + macStringified,
													"enforce has failed, sensor disconnected",
													[]int{0}, true})
											return
										} else {
											if globals.DebugActive {
												fmt.Printf("Sensor %v failed ID enforce\n", sensorDef.mac)
											}
											mlogger.Info(globals.SensorManagerLog,
												mlogger.LoggerData{"sensor " + macStringified,
													"enforce has failed",
													[]int{0}, true})
											continue
										}
									} else {
										continue
									}
								}

								// in case the sent ID and the defined ID is -1 is is set to 0xffff,
								// the sensor is considered not active
								if sensorDef.id == -1 && sensorDef.idSent == 65535 {
									if reject, newDevice, err := diskCache.MarkInvalidDevice([]byte(macStringified), globals.MaximumInvalidIDInternal); err == nil {
										if newDevice {
											// new device, we wait in case the id gets set
											if globals.DebugActive {
												fmt.Printf("Sensor %v has no valid id\n", sensorDef.mac)
											}
											mlogger.Info(globals.SensorManagerLog,
												mlogger.LoggerData{"sensor " + macStringified,
													"has no valid id yet",
													[]int{0}, true})
											// a delay is added to reduce activity on sensors with invalid ID's
											others.WaitRandom(globals.MaliciousTimeout)
										} else if reject {
											// the device has been unset for too long. We mark its MAC and disconnect
											if err := diskCache.RemoveInvalidDevice([]byte(macStringified)); err == nil {
												_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
												// A delay is inserted in case this is a malicious attempt
												others.WaitRandom(globals.MaliciousTimeout)
												return
											}
										}
									}
									continue
								}

								if err := diskCache.RemoveInvalidDevice([]byte(macStringified)); err != nil {
									return
								}

								// the strict condition on the ID can now be checked
								if sensorDef.idSent != sensorDef.id && sensorDef.strict {
									// sensor mismatch is considered illegal
									if globals.DebugActive {
										fmt.Printf("Sensor %v has been rejected\n", sensorDef.mac)
									}
									mlogger.Info(globals.SensorManagerLog,
										mlogger.LoggerData{"sensor " + macStringified,
											"rejected due to wrong id: " + strconv.Itoa(sensorDef.idSent),
											[]int{0}, true})
									_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
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
									if reject, newDevice, err := diskCache.MarkInvalidDevice([]byte(macStringified),
										globals.MaximumInvalidIDInternal); err == nil {
										if newDevice {
											// new device, we wait in case the id gets set
											if globals.DebugActive {
												fmt.Printf("Sensor %v:%v is not being used\n", sensorDef.mac, sensorDef.id)
											}
											mlogger.Info(globals.SensorManagerLog,
												mlogger.LoggerData{"sensor " + macStringified,
													"is not being used, ID: " + strconv.Itoa(sensorDef.id),
													[]int{0}, true})
											// a delay is added to reduce activity on sensors with invalid ID's
											others.WaitRandom(globals.MaliciousTimeout)
										} else if reject {
											// the device has been unset for too long. We mark its MAC and disconnect
											if err := diskCache.RemoveInvalidDevice([]byte(macStringified)); err == nil {
												_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
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
										err = diskCache.RemoveInvalidDevice([]byte(macStringified))
									}
								}()

								go func() {
									err := globals.SensorDBError
									for i := 0; i < 5 && err != nil; i++ {
										err = diskCache.RemoveSuspectedMAC([]byte(macStringified))
									}
								}()

								if err := diskCache.AddLookUp([]byte{byte(sensorDef.id)}, sensorDef.mac); err != nil {
									mlogger.Error(globals.SensorManagerLog,
										mlogger.LoggerData{"sensor " + macStringified,
											"failed to saving lookup declaration", []int{0}, true})
									return
								}

								if err := diskCache.MarkDeviceActive([]byte(macStringified)); err == nil {
									sensorDef.active = true
									ActiveSensors.Lock()
									ActiveSensors.Id[sensorDef.id] = sensorDef.mac
									ActiveSensors.Unlock()
									mlogger.Info(globals.SensorManagerLog,
										mlogger.LoggerData{"sensor " + macStringified,
											"is active with ID " + strconv.Itoa(sensorDef.id),
											[]int{0}, true})
									gateManager.SensorStructure.RLock()
									sensorDef.channels.gateChannel = gateManager.SensorStructure.DataChannel[sensorDef.id]
									gateManager.SensorStructure.RUnlock()
								}
							}

							// the sensor can now be considered valid and we send the data to the gate
							if sensorDef.channels.gateChannel == nil {
								// somehow sensor definition got corrupted
								if globals.DebugActive {
									fmt.Printf("Sensor %v has no valid gate associated\n", sensorDef.mac)
								}
								mlogger.Info(globals.SensorManagerLog,
									mlogger.LoggerData{"sensor " + macStringified,
										"has no valid gate associated",
										[]int{0}, true})
								return
							} else {
								// ATTENTION change this filtering in case the sensor sends more than just 255,1,-1
								//  can be done with flow := int(int8(data[2]))
								//  if math.Abs(float64(flow)) > maxValue then flow = 0
								flow := int(data[2])
								if flow == 255 {
									flow = -1
								} else if flow != 1 {
									flow = 0
								}
								sampleTS := time.Now().UnixNano()
								record := sensorDef.maxRate == 0 || lastSampleTS+sensorDef.maxRate < sampleTS
								if globals.GateMode {
									if !record {
										fmt.Printf("Sample arrived faster then %v ms and is skipped\n", sensorDef.maxRate/1000000)
									} else {
										if flow == 0 {
											fmt.Print("0")
										}
										fmt.Printf("\nSensor %v with id %v has sent %v at %v\n", macStringified, sensorDef.id, flow, sampleTS)
									}
								}
								if record {
									lastSampleTS = sampleTS
									for _, ch := range sensorDef.channels.gateChannel {
										ch <- dataformats.FlowData{
											Type:    "sensor",
											Name:    macStringified,
											Id:      sensorDef.id,
											Ts:      sampleTS,
											Netflow: flow,
										}
									}
								}
							}
							//gateManager.DistributeData(sensorDef.id, int(data[2]))

						}
					default:
						// we first check if this is a setID DoC attack
						if cmd[0] == CmdAPI["setid"].Cmd && maliciousSetIdDOS(ipc, macStringified) {
							// this is a malicious device
							// A delay is inserted in case this is a malicious attempt
							others.WaitRandom(globals.MaliciousTimeout)
							return
						}
						if sensorDef.channels.CmdAnswer == nil {
							// process is corrupted, we must terminate it
							if globals.DebugActive {
								fmt.Printf("sensorManager.sensorHandler: sensor commands channel found invalid\n")
							}
							mlogger.Error(globals.SensorManagerLog,
								mlogger.LoggerData{"sensorManager.sensorHandler",
									"critical error, sensor commands channel found invalid",
									[]int{0}, false})
							return
						}
						// this is a command answer
						// only the answer to setid can be allowed when the sensor is not active and id!=idSent
						if sensorDef.active ||
							sensorDef.id != sensorDef.idSent && cmd[0] == CmdAPI["setid"].Cmd {

							// we verify that we received a command answer from an active device
							if cmdLength, ok := CmdAnswerLen[cmd[0]]; ok {
								// in case if no CRC the length needs to be decrease
								if !globals.CRCused {
									cmdLength -= 1
								}
								// check if command answer is fully correct and forward it to the command process
								if cmdLength == 0 {
									//this can only happen when CRC is not used
									select {
									case sensorDef.channels.CmdAnswer <- cmd:
										select {
										case ans := <-sensorDef.channels.CmdAnswer:
											if ans != nil {
												if globals.MalicioudMode > globals.OFF {
													sensorDef.failures += 1
													if sensorDef.failures > globals.FailureThreshold {
														_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
														// A delay is inserted in case this is a malicious attempt
														others.WaitRandom(globals.MaliciousTimeout)
														return
													}
												}
											}
										case <-time.After(time.Duration(globals.SensorTimeout*3) * time.Second):
											if globals.DebugActive {
												fmt.Printf("sensorManager.sensorHandler: hanging operation in receiving command answer\n")
											}
											return
										}

									case <-time.After(time.Duration(globals.SensorTimeout*3) * time.Second):
										if globals.DebugActive {
											fmt.Printf("sensorManager.sensorHandler: hanging operation in sending "+
												"command answer %cmdLength\n", cmd)
										}
										return
									}
								} else {
									cmdd := make([]byte, cmdLength)
									if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
										deadlineFailed(macStringified, e)
										return
									}
									if _, e := conn.Read(cmdd); e != nil {
										failedRead(macStringified, ipc, e)
										// in case of malicious mode severe we flag the mac and the IP
										if globals.MalicioudMode > globals.OFF {
											sensorDef.failures += 1
											if sensorDef.failures > globals.FailureThreshold {
												_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
												// A delay is inserted in case this is a malicious attempt
												others.WaitRandom(globals.MaliciousTimeout)
												return
											}
										}
									} else {
										cmd = append(cmd, cmdd...)
										if globals.CRCused {
											crc := codings.Crc8(cmd[:len(cmd)-1])
											if crc != cmd[len(cmd)-1] {
												if globals.DebugActive {
													fmt.Print("sensorManager.sensorHandler: wrong CRC on received command answer\n")
												}
												mlogger.Info(globals.SensorManagerLog,
													mlogger.LoggerData{"sensor " + macStringified,
														"wrong CRC on received command answer",
														[]int{0}, true})
												// with a wrong CRC the message is rejected but the connection is not closed
												if globals.CRCMaliciousCount {
													// in case of malicious mode severe we flag the mac and the IP
													if globals.MalicioudMode > globals.OFF {
														sensorDef.failures += 1
														if sensorDef.failures > globals.FailureThreshold {
															_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
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
															_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
															// A delay is inserted in case this is a malicious attempt
															others.WaitRandom(globals.MaliciousTimeout)
															return
														}
													}
												}
												// in case we receive a valid answer to setid, we close the channel
												// this allows for the server to adapt to the new ID
												if cmd[0] == CmdAPI["setid"].Cmd {
													_ = diskCache.RemoveSuspectedMAC([]byte(sensorDef.mac))
													return
												}
											case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
												if globals.DebugActive {
													fmt.Printf("sensorManager.sensorHandler: hanging operation in receiving command answer\n")
												}
												return
											}
										case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
											// internal issue, all goroutines will close on time out including the channel
											if globals.DebugActive {
												fmt.Printf("sensorManager.sensorHandler: hanging operation in sending "+
													"command answer %cmdLength\n", cmd)
											}
										}
									}
								}
								_ = diskCache.RemoveSuspectedMAC([]byte(sensorDef.mac))
							} else {
								// illegal command answer received
								if globals.MalicioudMode > globals.OFF {
									sensorDef.failures += 1
									if sensorDef.failures > globals.FailureThreshold {
										_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
										// A delay is inserted in case this is a malicious attempt
										others.WaitRandom(globals.MaliciousTimeout)
										return
									}
								}
							}
						} else {
							// command answer received from a non valid device
							if globals.DebugActive {
								fmt.Printf("sensorManager.sensorHandler: device %v//%v not active and sending non-data packages\n", ipc, macStringified)
							}
							if globals.MalicioudMode > globals.OFF {
								sensorDef.failures += 1
								if sensorDef.failures > globals.FailureThreshold {
									_, _ = diskCache.MarkMAC([]byte(macStringified), globals.MaliciousTriesMac)
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
}
