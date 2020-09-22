package sensorManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
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

// TODO check if we keep stuff as is in memory or not

func handler(conn net.Conn) {

	mac := make([]byte, 6) // received amc address
	var sensorDef SensorDefinition

	// cleaning up at closure
	defer func() {

		// We close the channel and update the sensor definition entry, when applicable
		conn.Close()
		if sensorDef.Mac != "" {
			DeclaredSensors.Lock()
			sensorDef.CurrentChannel = nil
			DeclaredSensors.Mac[sensorDef.Mac] = sensorDef
			DeclaredSensors.Unlock()
		}

	}()

	// TODO we better store it
	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]

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
		// A delay is inserted in case this is a malicious attempt
		time.Sleep(time.Duration(globals.SensorTimeout) * time.Second)
	} else {
		mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", "", -1), " ")

		// sensor configuration data is retrieved and it is verified that no channel is already open
		DeclaredSensors.Lock()
		sensorDef = DeclaredSensors.Mac[mach]

		// TODO add limit on sensorDef.SuspectedConnection to disable the sensor and add it to a blocked list

		if sensorDef.CurrentChannel != nil {
			DeclaredSensors.Unlock()
			// The sensor has already an assigned TCP channel
			// We wait to see if it closes, if not the new connection channel is closed and marked as a possible attack
			time.Sleep(time.Duration(globals.SensorTimeout) * time.Second)
			DeclaredSensors.Lock()
			sensorDef = DeclaredSensors.Mac[mach]
			if sensorDef.CurrentChannel != nil {
				sensorDef.SuspectedConnection += 1
				DeclaredSensors.Mac[sensorDef.Mac] = sensorDef
				DeclaredSensors.Unlock()
				sensorDef.Mac = ""
				mlogger.Warning(globals.SensorManagerLog,
					mlogger.LoggerData{"sensor " + mach,
						"suspected malicious connection",
						[]int{1}, true})
				// We wait assuming it is an attack to slow it down
				wait := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(globals.MaliciousTimeout)
				time.Sleep(time.Duration(wait) * time.Second)
				// TODO store in a list of suspected devices/ip
				return
			}
		}
		// the sensor is returned and this is not suspected to be a malicious attack
		sensorDef.CurrentChannel = conn
		DeclaredSensors.Mac[sensorDef.Mac] = sensorDef
		DeclaredSensors.Unlock()

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

		// TODO HERE store the device to a device pending list

		fmt.Printf("%+v\n", sensorDef)
	}

	for {
		time.Sleep(36 * time.Hour)
	}
}
