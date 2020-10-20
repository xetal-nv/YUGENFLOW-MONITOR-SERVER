package exportManager

import (
	"encoding/json"
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"os/exec"
)

func customScripting(rst chan interface{}, chActuals, chReferences chan dataformats.MeasurementSample) {
	//cmd := exec.Command("python", "test.py")
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	//err := cmd.Run()
	//if err != nil {
	//	fmt.Printf("cmd.Run() failed with %s\n", err)
	//}
finished:
	for {
		select {
		case <-rst:
			mlogger.Info(globals.ExportManager,
				mlogger.LoggerData{"exportManager.customScripting",
					"service stopped",
					[]int{0}, true})
			fmt.Println("Closing exportManager.customScripting")
			rst <- nil
			break finished
		case data := <-chActuals:
			// TODO
			if encodedData, err := json.Marshal(data); err == nil {
				cmd := exec.Command("python", "test.py ", string(encodedData))
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					fmt.Printf("cmd.Run() failed with %s\n", err)
				}
				//fmt.Printf("actual %+v\n", string(encodedData))
			}
		case data := <-chReferences:
			// TODO
			fmt.Printf("reference %+v\n", data)
		}
	}
}
