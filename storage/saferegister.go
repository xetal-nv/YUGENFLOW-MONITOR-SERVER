package storage

import (
	"gateserver/support"
)

// SafeReg implement a single input (n) single output (o)
// non-blocking register. It blocks only when it is not initialised
func SafeReg(in, out chan interface{}, init ...interface{}) {
	var data interface{}
	r := func() {
		if len(init) != 1 {
			data = <-in
		} else {
			data = init
		}
		for {
			select {
			case data = <-in:
				//fmt.Println("received:", data)
			case out <- data:
				//fmt.Println("sent:", data)
			}
		}
	}
	go support.RunWithRecovery(r, nil)
}
