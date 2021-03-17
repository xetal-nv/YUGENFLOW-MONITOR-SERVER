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

func customScripting(rst chan interface{}, chActual chan dataformats.MeasurementSampleWithFlows, chReferences chan dataformats.MeasurementSample) {

finished:
	for {
		var err error
		var encodedData []byte
		select {
		case <-rst:
			fmt.Println("Closing exportManager.customScripting")
			time.Sleep(time.Duration(globals.SettleTime) * time.Second)
			rst <- nil
			break finished
		case data := <-chActual:
			encodedData, err = json.Marshal(data)
		case data := <-chReferences:
			encodedData, err = json.Marshal(data)
		}

		if err == nil {
			if globals.DebugActive {
				fmt.Printf("Export JSON: %v\n", strings.Replace(string(encodedData), "\"", "'", -1))
			}

			if globals.ExportAsync {
				cmd := exec.Command(globals.ExportActualCommand, globals.ExportActualArgument,
					strings.Replace(string(encodedData), "\"", "'", -1))
				if globals.ExportActualArgument == "" {
					cmd = exec.Command(globals.ExportActualCommand, strings.Replace(string(encodedData), "\"", "'", -1))
				}
				if globals.DebugActive {
					fmt.Println(cmd)
				}
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
