package gateManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"sync"
	"time"
)

/*
	set up the sensor2gates channels and relative gate services
	send data to the relevant entries
	save gate data in the database
*/

func Start(sd chan bool) {
	var err error

	if globals.DeviceManagerLog, err = mlogger.DeclareLog("yugenflow_gateManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_gateManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.DeviceManagerLog, 40, 30, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.DeviceManagerLog,
		mlogger.LoggerData{"gateManager.Start",
			"service started",
			[]int{0}, true})

	var rstC []chan bool
	for i := 0; i < 0; i++ {
		rstC = append(rstC, make(chan bool))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan bool) {
		<-sd
		fmt.Println("Closing gateManager")
		var wg sync.WaitGroup
		for _, ch := range rstC {
			wg.Add(1)
			go func(ch chan bool) {
				ch <- true
				select {
				case <-ch:
				case <-time.After(2 * time.Second):
				}
				wg.Done()
			}(ch)
		}
		wg.Wait()
		mlogger.Info(globals.DeviceManagerLog,
			mlogger.LoggerData{"gateManager.Start",
				"service stopped",
				[]int{0}, true})
		time.Sleep(time.Duration(globals.ShutdownTime) * time.Second)
		sd <- true
	}(sd, rstC)

	//recovery.RunWith(
	//	func() { ApiManager(rstC[0]) },
	//	func() {
	//		mlogger.Recovered(globals.DeviceManagerLog,
	//			mlogger.LoggerData{"clientManager.ApiManager",
	//				"ApiManager service terminated and recovered unexpectedly",
	//				[]int{1}, true})
	//	})

	for {
		time.Sleep(36 * time.Hour)
	}
}
