package exportManager

import (
	"encoding/json"
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os/exec"
	"strings"
	"time"
)

func customScripting(rst chan interface{}, chActual, chReferences chan dataformats.MeasurementSample) {
	var actualData = false
	snapshots := make(map[string]dataformats.MeasurementSample)
finished:
	for {
		var data dataformats.MeasurementSample
		select {
		case <-rst:
			fmt.Println("Closing exportManager.customScripting")
			time.Sleep(time.Duration(globals.SettleTime) * time.Second)
			rst <- nil
			break finished
		case data = <-chActual:
			actualData = true
		case data = <-chReferences:
			actualData = false
		}
		// TODO adding flow accumulation for actual data only
		if actualData {
			// in and out flows are accumulated only for actual data
			if pertinentSnapshot, ok := snapshots[data.Qualifier]; !ok {
				// we need to copy the data
				pertinentSnapshot.Ts = data.Ts
				pertinentSnapshot.Val = data.Val
				pertinentSnapshot.Qualifier = data.Qualifier
				pertinentSnapshot.Space = data.Space
				// TODO duplicate all flows
				snapshots[data.Qualifier] = pertinentSnapshot
			} else {
				//we need to update all values and accumulate flows
				// in case of overflow we mark the timestamp and rebase the in/out floes numbers
			}
		}
		if encodedData, err := json.Marshal(data); err == nil {
			if globals.DebugActive {
				fmt.Printf("Export JSON: %v\n", strings.Replace(string(encodedData), "\"", "'", -1))
			}
			if globals.ExportAsync {
				cmd := exec.Command(globals.ExportActualCommand, globals.ExportActualArgument,
					strings.Replace(string(encodedData), "\"", "'", -1))
				err := cmd.Start()
				if err != nil {
					// script report error
					if globals.DebugActive {
						fmt.Println("Export script has failed:", err.Error())
					}
					mlogger.Error(globals.ExportManagerLog,
						mlogger.LoggerData{Id: "exportManager.customScripting",
							Message: "error exporting data ",
							Data:    []int{1}, Aggregate: true})
				}
			} else {
				cmd, err := exec.Command(globals.ExportActualCommand, globals.ExportActualArgument,
					strings.Replace(string(encodedData), "\"", "'", -1)).Output()
				if err != nil || len(cmd) != 0 {
					// script report error
					if globals.DebugActive {
						if err != nil {
							fmt.Println("Export script has failed:", err.Error())
						} else {
							fmt.Println("Export script reported failure:", string(cmd))
						}
					}
					mlogger.Error(globals.ExportManagerLog,
						mlogger.LoggerData{Id: "exportManager.customScripting",
							Message: "error exporting data ",
							Data:    []int{1}, Aggregate: true})
				}
			}
		}
	}
}
