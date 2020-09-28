package sensorManager

import (
	"fmt"
	"gateserver/supp"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
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
					if globals.ResetSlot == "" {
						// TODO reset
						//  will try to reset every day in a given interval all sensors that are
						//  in ActiveSensors and marked as active in sensorDB

					}
				}
			}
		}
	}
	if globals.DebugActive {
		fmt.Println(globals.ResetSlot, globals.ResetPeriod)
		fmt.Println("*** WARNING: sensor reset is disabled ***")
		os.Exit(0)
	}
}
