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
			start := true
			for start {
				select {
				case data = <-in:
					start = false
				case out <- nil:
					//fmt.Println("nil")
				}
			}
			//data = <-in
		} else {
			data = init[0]
		}
		//fmt.Println("end of nil")
		//fmt.Println(data)
		for {
			select {
			case data = <-in:
				//fmt.Println(data)
			case out <- data:
			}
		}
	}
	go support.RunWithRecovery(r, nil)
}
