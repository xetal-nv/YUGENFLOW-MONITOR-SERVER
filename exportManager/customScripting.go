package exportManager

import (
	"encoding/json"
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"os"
	"os/exec"
	"strings"
	"time"
)

func customScripting(rst chan interface{}, chActuals, chReferences chan dataformats.MeasurementSample) {
finished:
	for {
		var data dataformats.MeasurementSample
		select {
		case <-rst:
			fmt.Println("Closing exportManager.customScripting")
			time.Sleep(time.Duration(globals.SettleTime) * time.Second)
			rst <- nil
			break finished
		case data = <-chActuals:
		case data = <-chReferences:
		}
		// TODO: add async, handle answer in debug, log error
		if encodedData, err := json.Marshal(data); err == nil {
			//stringedEncodedData := strings.Replace(string(encodedData),"\"", "'", -1)
			cmd := exec.Command("python", "test.py ",
				strings.Replace(string(encodedData), "\"", "'", -1))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				fmt.Printf("cmd.Run() failed with %s\n", err)
			}
		}
	}
}
