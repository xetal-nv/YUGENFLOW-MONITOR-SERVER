package sensorManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/dbs/sensorDB"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

/*
	initiate the TCP channels with all checks
	send data using gateManager channels to the proper gates
*/

func handler(conn net.Conn) {

	mac := make([]byte, 6) // received amc address
	var sensorDef sensorDefinition

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
		if globals.DebugActive {
			fmt.Printf("sensorManager.handler: error on setting deadline for %v : %v\n", ipc, e)
		}
		mlogger.Error(globals.SensorManagerLog,
			mlogger.LoggerData{"sensor " + ipc,
				"error on setting deadline: " + e.Error(),
				[]int{}, false})
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
		_, _ = sensorDB.AddIP([]byte(ipc), globals.MaliciousTriesIP)
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
			def, erDB = sensorDB.ReadDefinition([]byte("default"))
			if erDB != nil {
				// this should never happen
				if globals.DebugActive {
					fmt.Println("sensor ", ipc, "read sensorDB error:", erDB.Error())
				}
				mlogger.Error(globals.SensorManagerLog,
					mlogger.LoggerData{"sensor " + ipc,
						"error reading sensorDB :" + e.Error(),
						[]int{1}, true})
				return
			}
		}
		sensorDef = sensorDefinition{
			mac:     mach,
			id:      def.Id,
			bypass:  def.Bypass,
			report:  def.Report,
			enforce: def.Enforce,
			strict:  def.Strict,
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
				_, _ = sensorDB.AddIP([]byte(ipc), globals.MaliciousTriesIP)
				return
			}
		}
		// the sensor is returned and this is not suspected to be a malicious attack
		sensorDef.channels = SensorChannel{
			Tcp:     conn,
			Process: make(chan dataformats.SensorCommand, globals.ChannellingLength),
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
			if globals.DebugActive {
				fmt.Printf("sensorManager.handler: timeout reset error %v with device %v//%v\n", e.Error(), ipc, mach)
			}
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensor " + mach,
					"failed to reset deadline timeouts",
					[]int{}, false})
		}

		// TODO make core loop for data read, command send and command answer read

		fmt.Printf("%+v\n", sensorDef)
		fmt.Println(sensorDB.LookUpMac([]byte(strconv.Itoa(sensorDef.id))))
	}

	for {
		time.Sleep(36 * time.Hour)
	}
}
