package exportManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"time"
	"xetal.ddns.net/utils/recovery"
)

func Start(sd chan bool) {

	var err error

	if globals.ExportManagerLog, err = mlogger.DeclareLog("yugenflow_exportManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_exportManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.ExportManagerLog, 50, 50, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.ExportManagerLog,
		mlogger.LoggerData{"exportManager.Start",
			"service started",
			[]int{0}, true})

	var rstC []chan interface{}
	for i := 0; i < 1; i++ {
		rstC = append(rstC, make(chan interface{}))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan interface{}) {
		<-sd
		fmt.Println("Closing exportManager")
		for _, ch := range rstC {
			ch <- nil
			select {
			case <-ch:
			case <-time.After(time.Duration(globals.SettleTime) * time.Second):
			}
		}
		mlogger.Info(globals.ExportManagerLog,
			mlogger.LoggerData{"exportManager.Start",
				"service stopped",
				[]int{0}, true})
		time.Sleep(time.Duration(globals.SettleTime) * time.Second)
		sd <- true
	}(sd, rstC)

	ExportActuals = make(chan dataformats.MeasurementSampleWithFlows, globals.ChannellingLength)
	ExportReference = make(chan dataformats.MeasurementSample, globals.ChannellingLength)

	recovery.RunWith(
		func() {
			customScripting(rstC[0], ExportActuals, ExportReference)
		},
		func() {
			mlogger.Recovered(globals.ExportManagerLog,
				mlogger.LoggerData{"exportManager.customScripting",
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		})

}
