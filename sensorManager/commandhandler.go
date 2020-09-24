package sensorManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
)

func sensorCommand(channels SensorChannel, mac string) {
	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
			"service started",
			[]int{}, true})
	reset := channels.Reset
	command := channels.Commands
	//conn := channels.Tcp
	if globals.DebugActive {
		fmt.Println("sensorManager.sensorCommand: started", mac)
	}
finished:
	for {
		select {
		case <-reset:
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
					"service stopped",
					[]int{}, true})
			break finished
		case cmd := <-command:
			// TODO everything
			println("execute %x fpr %x\n", cmd, mac)
		}
	}
	if globals.DebugActive {
		fmt.Println("sensorManager.sensorCommand: ended", mac)
	}
	reset <- true
}
