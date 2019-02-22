package storage

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
				if support.Debug != 3 && support.Debug != 4 {
					if err := StoreSample(&SerieSample{id, d.Ts, d.Ct}, !support.Stringending(id, "current")); err != nil {
						log.Printf("storage.TimedIntDBS: DBS handler %v error %v\n", id, err)
					}
				}
				if support.Debug > 0 {
					fmt.Println("DEBUG DBS id:", id, "got data", d, "is current", support.Stringending(id, "current"))
				}
			case a := <-rst:
				// Reset via API might be dangerous, this is justa  placeholder
				log.Println("storage.TimedIntDBS: handler id:", id, "got reset request", a, " - does nothing still")
			}
		}
	}
	er := func() {
		log.Printf("storage.TimedIntDBS: recovering from crash\n ")
	}
	go support.RunWithRecovery(r, er)
}

func handlerDBS(id string, in chan interface{}, rst chan bool, a SampleData) {

	r := func() {
		log.Printf("register.TimedIntDBS: DBS handler %v started\n", id)
		for {
			select {
			case d := <-in:
				// TODO to be fully tested with the API server
				if support.Debug != 3 && support.Debug != 4 {
					if e := a.Extract(d); e == nil {
						if err := StoreSample(a, !support.Stringending(id, "current")); err != nil {
							log.Printf("storage.TimedIntDBS: DBS handler %v error %v\n", id, err)
						}
					} else {
						log.Println(e.Error(), d)
					}
				}
				if support.Debug > 0 {
					fmt.Println("DEBUG DBS id:", id, "got data", d, "is current", support.Stringending(id, "current"))
				}
			case a := <-rst:
				// Reset via API might be dangerous, this is justa  placeholder
				log.Println("storage.TimedIntDBS: handler id:", id, "got reset request", a, " - does nothing still")
			}
		}
	}
	er := func() {
		log.Printf("storage.TimedIntDBS: recovering from crash\n ")
	}
	go support.RunWithRecovery(r, er)
}
