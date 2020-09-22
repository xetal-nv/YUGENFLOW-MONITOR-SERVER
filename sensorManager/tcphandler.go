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

// TODO change using bolt for most data

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
		}

	}()

	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]
	// the IP is checked in the disabled list
	// TODO move to bolt

	//MaliciousIPS.RLock()
	//if MaliciousIPS.Disabled[ipc] {
	//	MaliciousIPS.RUnlock()
	//	// We wait assuming it is an attack to slow it down
	//	wait := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(globals.MaliciousTimeout)
	//	time.Sleep(time.Duration(wait) * time.Second)
	//	mlogger.Warning(globals.SensorManagerLog,
	//		mlogger.LoggerData{"device " + ipc,
	//			"malicious, connection refused",
	//			[]int{1}, true})
	//	return
	//}
	//MaliciousIPS.RUnlock()

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

		// read sensor definition form the sensor DB and srore it locally
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

		// TODO make suspected ip and mac (store number of suspected activities)

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
				// We wait assuming it is an attack to slow it down
				wait := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(globals.MaliciousTimeout)
				time.Sleep(time.Duration(wait) * time.Second)
				return
			}
		}
		// the sensor is returned and this is not suspected to be a malicious attack
		sensorDef.channels = SensorChannel{
			Tcp:     conn,
			Process: make(chan dataformats.SensorCommand),
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

		// TODO HERE store the device to a device pending list

		fmt.Printf("%+v\n", sensorDef)
		fmt.Println(sensorDB.LookUpMac([]byte(strconv.Itoa(sensorDef.id))))
	}

	for {
		time.Sleep(36 * time.Hour)
	}
}
