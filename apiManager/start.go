package apiManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"time"
	"xetal.ddns.net/utils/recovery"
)

func Start(sd chan bool) {
	var err error

	if globals.ApiManagerLog, err = mlogger.DeclareLog("yugenflow_apiManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_apiManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.ApiManagerLog, 80, 30, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.ApiManagerLog,
		mlogger.LoggerData{"apiManager.Start",
			"service started",
			[]int{1}, true})

	var rstC []chan bool
	for i := 0; i < 1; i++ {
		rstC = append(rstC, make(chan bool))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan bool) {
		<-sd
		fmt.Println("Closing apiManager")
		for _, ch := range rstC {
			ch <- true
			select {
			case <-ch:
			case <-time.After(2 * time.Second):
			}
		}
		mlogger.Info(globals.ApiManagerLog,
			mlogger.LoggerData{"apiManager.Start",
				"service stopped",
				[]int{1}, true})
		time.Sleep(3 * time.Second)
		sd <- true
	}(sd, rstC)

	recovery.RunWith(
		func() { ApiManager(rstC[0]) },
		func() {
			mlogger.Recovered(globals.ApiManagerLog,
				mlogger.LoggerData{"clientManager.ApiManagerLog",
					"ApiManagerLog service terminated and recovered unexpectedly",
					[]int{1}, true})
		})
}
