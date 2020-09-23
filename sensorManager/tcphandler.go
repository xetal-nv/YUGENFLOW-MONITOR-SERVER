package sensorManager

import (
	"fmt"
	"gateserver/codings"
	"gateserver/dataformats"
	"gateserver/dbs/sensorDB"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
)

/*
	initiate the TCP channels with all checks
	send data using gateManager channels to the proper gates
*/

func handler(conn net.Conn) {

	// support methods
	deadlineFailed := func(ipc string, e error) {
		if globals.DebugActive {
			fmt.Printf("sensorManager.handler: error on setting deadline for %v : %v\n", ipc, e)
		}
		mlogger.Error(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + ipc,
				"error on setting deadline: " + e.Error(),
				[]int{}, false})
	}
	failedRead := func(mach, ipc string, e error) {
		if globals.DebugActive {
			log.Printf("sensorManager.handler: error reading from %v//%v : %v\n", ipc, mach, e)
		}
		mlogger.Error(globals.SensorManagerLog,
			mlogger.LoggerData{mach + " read error",
				"ip: " + ipc,
				[]int{1}, true})
	}

	var sensorDef sensorDefinition
	mac := make([]byte, 6) // received amc address

	// cleaning up at closure
	defer func() {

		// We close the channel and update the sensor definition entry, when applicable
		_ = conn.Close()
		if sensorDef.mac != "" {
			ActiveSensors.Lock()
			delete(ActiveSensors.Id, sensorDef.id)
			delete(ActiveSensors.Mac, sensorDef.mac)
			ActiveSensors.Unlock()
			_ = sensorDB.DeleteDevice([]byte(sensorDef.mac))
		}

	}()

	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]
	// the IP is checked in the disabled list
	if banned, err := sensorDB.CheckIP([]byte(ipc), globals.MaliciousTriesIP); err == nil && banned {
		// We wait assuming it is an attack to slow it down
		wait := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(globals.MaliciousTimeout)
		time.Sleep(time.Duration(wait) * time.Second)
		mlogger.Warning(globals.SensorManagerLog,
			mlogger.LoggerData{"device " + ipc,
				"malicious, connection refused",
				[]int{1}, true})
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
		mlogger.Error(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + ipc,
				"error reading MAC :" + e.Error(),
				[]int{1}, true})
		// A delay is inserted in case this is a malicious attempt and we mark the IP as suspicious
		time.Sleep(time.Duration(globals.SensorTimeout) * time.Second)
		_, _ = sensorDB.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
		return
	} else {
		mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", "", -1), " ")
		// the mac is checked in the disabled list
		if banned, err := sensorDB.CheckMAC([]byte(mach), globals.MaliciousTriesMac); err == nil && banned {
			// We wait assuming it is an attack to slow it down
			wait := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(globals.MaliciousTimeout)
			time.Sleep(time.Duration(wait) * time.Second)
			mlogger.Warning(globals.SensorManagerLog,
				mlogger.LoggerData{"device " + mach,
					"malicious, connection refused",
					[]int{1}, true})
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
			mac:     mach,
			id:      def.Id,
			bypass:  def.Bypass,
			report:  def.Report,
			enforce: def.Enforce,
			strict:  def.Strict,
			accept:  !def.Bypass && !def.Report && !def.Enforce && !def.Strict,
			active:  false,
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
				mlogger.Warning(globals.SensorManagerLog,
					mlogger.LoggerData{"sensor " + mach,
						"suspected malicious connection",
						[]int{1}, true})
				// We wait assuming it is an attack to slow it down and mark the IP as suspicious
				wait := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(globals.MaliciousTimeout)
				time.Sleep(time.Duration(wait) * time.Second)
				_, _ = sensorDB.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
				return
			}
		}
		// the sensor is returned and this is not suspected to be a malicious attack
		sensorDef.channels = SensorChannel{
			Tcp:     conn,
			Process: make(chan dataformats.CommandAnswer, globals.ChannellingLength),
		}

		ActiveSensors.Mac[sensorDef.mac] = sensorDef.channels
		ActiveSensors.Unlock()

		// if enabled, the EEPROM is refreshed
		if globals.SensorEEPROMResetEnabled {
			if e := setSensorParameters(conn, mach); e != nil {
				if globals.DebugActive {
					fmt.Printf("sensorManager.handler: closing TCP channel to %v//%v on "+
						"EEPROM refresh error : %v\n", ipc, mach, e)
				}
				mlogger.Error(globals.SensorManagerLog,
					mlogger.LoggerData{"sensor " + mach,
						"error refreshing EEPROM : " + e.Error(),
						[]int{1}, true})
				return
			}
		}

		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + mach,
				"refreshing EEPROM successful",
				[]int{1}, true})

		if err := sensorDB.MarkDeviceNotActive([]byte(mach)); err != nil {
			mlogger.Error(globals.SensorManagerLog,
				mlogger.LoggerData{"sensor " + mach,
					"failed to add to device list",
					[]int{1}, true})
			return
		}

		if e := conn.SetDeadline(time.Time{}); e != nil {
			deadlineFailed(ipc, e)
		}

		if globals.DebugActive {
			fmt.Printf("Sensor Definition: %+v\n", sensorDef)
		}

		loop := true
		for loop {
			cmd := make([]byte, 1)
			if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
				deadlineFailed(ipc, e)
				return
			}
			if _, e := conn.Read(cmd); e != nil {
				if e == io.EOF {
					// in case of channel closed (EOF) it gets logged and the handler terminated
					if globals.DebugActive {
						fmt.Printf("sensorManager.handler: connection lost with device %v//%v\n", ipc, mach)
					}
					mlogger.Error(globals.SensorManagerLog,
						mlogger.LoggerData{mach + " connection lost",
							"ip: " + ipc,
							[]int{1}, true})
					loop = false
				} else {
					failedRead(mach, ipc, e)
				}
				loop = false
			} else {
				// TODO Core part, based on the existing server
				switch cmd[0] {
				case 1:
					//fmt.Println("ok")
					//loop = false
					// this is a data packet
					var data []byte
					if globals.CRCused {
						data = make([]byte, 4)
					} else {
						data = make([]byte, 3)
					}
					if e := conn.SetDeadline(time.Now().Add(time.Duration(globals.TCPdeadline) * time.Hour)); e != nil {
						deadlineFailed(ipc, e)
						return
					}
					if _, e := conn.Read(data); e != nil {
						failedRead(mach, ipc, e)
						// A delay is inserted in case this is a malicious attempt
						time.Sleep(time.Duration(globals.SensorTimeout) * time.Second)
						// in case of malicious mode severe we flag the mac
						if globals.MalicioudMode == globals.SEVERE {
							_, _ = sensorDB.MarkMAC([]byte(mach), globals.MaliciousTriesMac)
						}
						loop = false
					} else {
						// valid data
						valid := true
						if globals.CRCused {
							msg := append(cmd, data[:3]...)
							crc := codings.Crc8(msg)
							if crc != data[3] {
								if globals.DebugActive {
									fmt.Print("servers.handlerTCPRequest: wrong CRC on received message\n")
								}
								valid = false
							}
						}
						if valid {
							// data is valid, we flag the device as active if needed
							if !sensorDef.active {
								if err := sensorDB.MarkDeviceActive([]byte(mach)); err == nil {
									sensorDef.active = true
								}
							}
							deviceId := int(data[1]) | int(data[0])<<8
							if globals.DebugActive {
								fmt.Printf("Valid data (%v) for device ID %v\n", valid, deviceId)
							}
							// TODO HERE what happens here depends on the sensor ID flags
							fmt.Println("id's", sensorDef.id, deviceId)
							loop = false
						}
					}
				default:
					fmt.Println("not ok")
					loop = false
					// this is a command answer
				}
			}
		}

	}

}
