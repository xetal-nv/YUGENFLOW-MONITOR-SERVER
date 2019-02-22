package storage

import (
	"countingserver/support"
)

// IntCell implement a single input (n) single output (o)
// non-blocking integer register.
// d is its default value at start, if not given -1 will be used
func IntCell(_ string, in, out chan int, d ...int) { // id
	r := func() {
		var data int
		if len(d) == 1 {
			data = d[0]
		} else {
			data = -1
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
