package registers

import (
	"playground/support"
	"strconv"
)

// IntCell implement a single input (n) single output (o)
// non-blocking integer register.
// d is its default value at start, if not given -1 will be used
func IntCell(id string, n, o chan int, d ...int) {
	r := func() {
		var data int
		if len(d) == 1 {
			data = d[0]
		} else {
			data = -1
		}
		for {
			select {
			case data = <-n:
			case o <- data:
			}
		}
	}
	go support.RunWithRecovery(r, nil)
}

func IntBank(id string, n, o []chan int, d ...int) bool {
	if len(n) != len(o) {
		return true
	}
	if len(d) != 0 {
		if len(d) != len(n) {
			return true
		} else {
			for i, r := range n {
				IntCell(id+strconv.Itoa(i), r, o[i], d[i])
			}
		}
	} else {
		for i, r := range n {
			IntCell(id+strconv.Itoa(i), r, o[i])
		}
	}
	return false
}
