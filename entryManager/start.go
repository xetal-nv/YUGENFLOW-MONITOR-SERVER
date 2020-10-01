package entryManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"sync"
	"time"
	"xetal.ddns.net/utils/recovery"
)

/*
	set up the sensor2gates channels and relative gate services
	send data to the relevant entries
	save gate data in the database
*/

func Start(sd chan bool) {
	var err error

	if globals.EntryManagerLog, err = mlogger.DeclareLog("yugenflow_entryManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_entryManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.EntryManagerLog, 40, 30, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.EntryManagerLog,
		mlogger.LoggerData{"entryManager.Start",
			"service started",
			[]int{0}, true})

	var rstC []chan interface{}
	for i := 0; i < 1; i++ {
		rstC = append(rstC, make(chan interface{}))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan interface{}) {
		<-sd
		fmt.Println("Closing gateManager")
		var wg sync.WaitGroup
		for _, ch := range rstC {
			wg.Add(1)
			go func(ch chan interface{}) {
				ch <- nil
				select {
				case <-ch:
				case <-time.After(time.Duration(globals.ShutdownTime) * time.Second):
				}
				wg.Done()
			}(ch)
		}
		wg.Wait()
		mlogger.Info(globals.EntryManagerLog,
			mlogger.LoggerData{"entryManager.Start",
				"service stopped",
				[]int{0}, true})
		time.Sleep(time.Duration(globals.ShutdownTime) * time.Second)
		sd <- true
	}(sd, rstC)

	recovery.RunWith(
		func() { entry(rstC[0]) },
		func() {
			mlogger.Recovered(globals.EntryManagerLog,
				mlogger.LoggerData{"entryManager.entry",
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		})

	//for {
	//	time.Sleep(36 * time.Hour)
	//}
}
