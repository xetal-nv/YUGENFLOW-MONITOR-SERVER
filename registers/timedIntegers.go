package registers

import (
	"fmt"
	"playground/support"
)

type DataCt struct {
	Ts int64 // timestamp
	Ct int   // counter
}

// IntCell implement a single input (n) single output (o)
// non-blocking integer register.
// d is its default value at start, if not given -1 will be used
func TimedIntCell(id string, in, out chan DataCt, d ...DataCt) {
	r := func() {
		var data DataCt
		if len(d) == 1 {
			data = d[0]
		} else {
			data.Ts = -1
			data.Ct = -1
		}
		for {
			select {
			case data = <-in:
			case out <- data:
			}
		}
	}
	go support.RunWithRecovery(r, nil)
}

// TODO need to be done with a fast key-value DBS!
// TODO HOW
// TODO use badger as a single database,
// TODO for every analysis and space create a first, step and last entry to use for storing and readin
// TODO every new sample will use as index a name plkus the last+ step and update last, not the timestamp received
// TODO if the timestamp received is different of more than 1 step, we will add a sam[;e equal to the previousone
// TODO if the database is started and the last is very different that the current time, we will fill the series with zeros
// TODO give the option to reset the database via API ?
// TODO for the current data we should use a separate database with limited life span of samples as from the .env file
func TimedIntDataBank(id string, in chan DataCt, rst chan bool) {
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
