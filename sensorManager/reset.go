package sensorManager

import (
	"fmt"
	"gateserver/supp"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"strings"
	"time"
)

func sensorReset(rst chan bool) {
	if globals.ResetSlot != "" {
		var start, stop time.Time
		period := strings.Split(globals.ResetSlot, " ")
		valid := false
		if v, e := time.Parse(supp.TimeLayout, strings.Trim(period[0], " ")); e == nil {
			start = v
			if v, e = time.Parse(supp.TimeLayout, strings.Trim(period[1], " ")); e == nil {
				stop = v
				valid = true
			}
		}
		if valid {
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.sensorReset",
					"service started",
					[]int{0}, true})
			if globals.DebugActive {
				fmt.Printf("*** INFO: sensor reset is set from %v tp %v\n", start, stop)
			}
			var macs []string
			var channels []SensorChannel
			for {
				select {
				case <-rst:
					fmt.Println("Closing sensorManager.sensorReset")
					mlogger.Info(globals.SensorManagerLog,
						mlogger.LoggerData{"sensorManager.sensorReset",
							"service stopped",
							[]int{0}, true})
					rst <- true
					return
				case <-time.After(time.Duration(globals.ResetPeriod) * time.Minute):
					// TODO HERE
					if doIt, e := supp.InClosureTime(start, stop); e == nil {
						//  then cycle among all devices till all are reset
						fmt.Println(doIt)
					} else {
						// error
						fmt.Println("error")
					}
					if macs == nil {
						ActiveSensors.RLock()
						for k, v := range ActiveSensors.Mac {
							macs = append(macs, k)
							channels = append(channels, v)

						}
						ActiveSensors.RUnlock()
						fmt.Println("NEW DEVICES TO BE RESET", macs, channels)
						// TODO reset
						//  will try to reset every day in a given interval all sensors that are
						//  in ActiveSensors and marked as active in sensorDB
					}
					//} else {
					//	if len(macs) > 1 {
					//		macs = macs[1:]
					//		channels = channels[1:]
					//		fmt.Println("DEVICES TO BE STILL RESET", macs, channels)
					//	} else {
					//		macs = nil
					//		channels = nil
					//	}
					//}
				}
			}
		}
	}
	if globals.DebugActive {
		fmt.Println(globals.ResetSlot, globals.ResetPeriod)
		fmt.Println("*** WARNING: sensor reset is disabled ***")
	}
}
