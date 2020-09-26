package sensorManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"time"
)

func sensorReset(rst chan bool) {
	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.sensorReset",
			"service started",
			[]int{}, true})
	for {
		select {
		case <-rst:
			fmt.Println("Closing sensorManager.sensorReset")
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.sensorReset",
					"service stopped",
					[]int{}, true})
			rst <- true
		case <-time.After(time.Hour):
			// TODO reset
			//  will try to reset every day in a given interval all sensors that are
			//  in ActiveSensors and marked as active in sensorDB

		}
	}
}
