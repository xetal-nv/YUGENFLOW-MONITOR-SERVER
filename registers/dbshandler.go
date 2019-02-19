package registers

import (
	"fmt"
	"log"
)

// TODO need to be done with a fast key-value DBS!
// TODO HOW
// TODO use badger as a single database,
// TODO for every analysis and space create a fromRst, step and lastUpdt entry to use for storing and readin
// TODO every new sample will use as index a name plkus the lastUpdt+ step and update lastUpdt, not the timestamp received
// TODO if the timestamp received is different of more than 1 step, we will add a sam[;e equal to the previousone
// TODO if the database is started and the lastUpdt is very different that the current time, we will fill the series with zeros
// TODO give the option to reset the database via API ?
// TODO for the current data we should use a separate database with limited life span of samples as from the .env file
func TimedIntDBS(id string, in chan DataCt, rst chan bool) {
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				log.Printf("registers.TimedIntDBS: recovering from crash\n ")
				go TimedIntDBS(id, in, rst)
			}
		}
	}()
	fmt.Println("DBS id:", id, "called TBD")
	for {
		select {
		case d := <-in:
			fmt.Println("DBS id:", id, "got data", d)
		case a := <-rst:
			fmt.Println("DBS id:", id, "got reset request", a, " TBD")
		}
	}
}
