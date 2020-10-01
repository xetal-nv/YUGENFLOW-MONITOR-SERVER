package entryManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
)

func entry(rst chan interface{}) {
	// TODO everything, must include duping data as well

	<-rst
	fmt.Println("Closing entryManager.entry")
	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"entryManager.entry",
			"service stopped",
			[]int{0}, true})
	rst <- nil

}
