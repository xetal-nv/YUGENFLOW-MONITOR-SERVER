package sensorManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"log"
	"net"
	"strings"
	"time"
)

/*
	initiate the TCP channels with all checks
	send data using gateManager channels to the proper gates
*/

func handler(conn net.Conn) {
	println("being done!")

	defer conn.Close()
	mac := make([]byte, 6) // received amc address

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

		// once a valid MAC is received, the EEPROM is refreshed (if applicable)
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

		// TODO once the mac has been received we need to verify the sensor declaration
		DeclaredSensors.RLock()
		sensorDef := DeclaredSensors.Mac[mach]
		DeclaredSensors.RUnlock()
		fmt.Printf("%+v\n", sensorDef)
	}

	for {
		time.Sleep(36 * time.Hour)
	}
}
