package sensorManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
)

func sensorCommand(chs SensorChannel, mac string) {
	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
			"service started",
			[]int{}, true})
	if globals.DebugActive {
		fmt.Println("sensorManager.sensorCommand: started", mac)
	}
finished:
	for {
		select {
		case <-chs.Reset:
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
					"service stopped",
					[]int{}, true})
			break finished
		case cmd := <-chs.Commands:
			// TODO this is a command request
			fmt.Printf("execute %x fpr %v\n", cmd, mac)
		case ans := <-chs.CmdAnswer:
			// TODO this is an unsolicited command answer
			fmt.Printf("unexpected answer %x fpr %v\n", ans, mac)
		}
	}
	if globals.DebugActive {
		fmt.Println("sensorManager.sensorCommand: ended", mac)
	}
	chs.Reset <- true
}
