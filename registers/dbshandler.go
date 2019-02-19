package registers

import (
	"countingserver/support"
	"fmt"
	"log"
)

func TimedIntDBS(id string, in chan DataCt, rst chan bool) {

	r := func() {
		log.Printf("register.TimedIntDBS: DBS handler %v started\n", id)
		for {
			select {
			case d := <-in:
				// TODO to be fully tested with the API server
				if err := StoreSerieSample(id, d.Ts, d.Ct, !support.Stringending(id, "current")); err != nil {
					log.Printf("registers.TimedIntDBS: DBS handler %v error %v\n", id, err)
				}
				// DEBUG
				fmt.Println("DEBUG DBS id:", id, "got data", d, "is current", support.Stringending(id, "current"))
			case a := <-rst:
				// Reset via API might be dangerous, this is justa  placeholder
				log.Println("registers.TimedIntDBS: handler id:", id, "got reset request", a, " - does nothing still")
			}
		}
	}
	er := func() {
		log.Printf("registers.TimedIntDBS: recovering from crash\n ")
	}
	go support.RunWithRecovery(r, er)
}
