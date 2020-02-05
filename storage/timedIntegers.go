package storage

import (
	"gateserver/support"
)

type DataCt struct {
	Ts int64 // timestamp
	Ct int   // counter
}

// IntCell implement a single input (n) single output (o)
// non-blocking integer register.
// d is its default value at start, if not given -1 will be used
//noinspection GoUnusedFunction
func _TimedIntCell(_ string, in, out chan DataCt, d ...DataCt) { // is
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
