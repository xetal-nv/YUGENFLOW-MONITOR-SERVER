package storage

import (
	"gateserver/support"
	"log"
)

// SafeRegGeneric implement a generic single input (n) single output (o)
// non-blocking register. It blocks only when it is not initialised

func SafeRegGeneric(tag string, in, out chan interface{}, init ...interface{}) {
	var data interface{}
	r := func() {
		if len(init) != 1 {
			start := true
			for start {
				select {
				case data = <-in:
					start = false
				case out <- nil:
				}
			}
		} else {
			data = init[0]
			log.Printf("Register %v initialised with %v\n", tag, data)
		}
		// fmt.Println(tag, "started")
		for {
			// fmt.Println(tag, data)
			select {
			case data = <-in:
			case out <- data:
				// fmt.Println(tag, data)
			}
		}
	}
	go support.RunWithRecovery(r, nil)
}
