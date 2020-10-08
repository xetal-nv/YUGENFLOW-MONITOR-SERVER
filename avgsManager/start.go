package avgsManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"sync"
	"time"
	"xetal.ddns.net/utils/recovery"
)

func Start(sd chan bool) {
	var err error

	if globals.AvgsLogger, err = mlogger.DeclareLog("yugenflow_avgsManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_avgsManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.AvgsLogger, 50, 50, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.AvgsLogger,
		mlogger.LoggerData{"avgsManager.Start",
			"service started",
			[]int{0}, true})

	var rstC []chan interface{}
	for i := 0; i < 1; i++ {
		rstC = append(rstC, make(chan interface{}))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan interface{}) {
		<-sd
		fmt.Println("Closing avgsManager")
		var wg sync.WaitGroup
		for _, ch := range rstC {
			wg.Add(1)
			go func(ch chan interface{}) {
				ch <- nil
				select {
				case <-ch:
				case <-time.After(time.Duration(globals.SettleTime) * time.Second):
				}
				wg.Done()
			}(ch)
		}
		wg.Wait()
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"avgsManager.Start",
				"service stopped",
				[]int{0}, true})
		time.Sleep(time.Duration(globals.SettleTime) * time.Second)
		sd <- true
	}(sd, rstC)

	recovery.RunWith(
		func() { calculator(rstC[0]) },
		func() {
			mlogger.Recovered(globals.AvgsLogger,
				mlogger.LoggerData{"avgsManager.calculator",
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		})

	//for {
	//	time.Sleep(36 * time.Hour)
	//}
}
