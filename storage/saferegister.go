package storage

import (
	"gateserver/support"
	"log"
)

// SafeReg implement a single input (n) single output (o)
// non-blocking register. It blocks only when it is not initialised

func SafeReg(tag string, in, out chan interface{}, init ...interface{}) {
	var data interface{}
	r := func() {
		if len(init) != 1 {
			start := true
			for start {
				select {
				case data = <-in:
					start = false
				case out <- nil:
					//fmt.Println(tag,"register sent nil")
				}
			}
			//data = <-in
		} else {
			data = init[0]
			log.Printf("Register %v initialised with %v\n", tag, data)
		}
		//fmt.Println("end of nil")
		//fmt.Println(tag, data)
		for {
			//fmt.Println(tag, "register waiting")
			select {
			case data = <-in:
				//fmt.Println(tag, "register got data",data)
			case out <- data:
				//fmt.Println(tag, "register sent data",data)
			}
		}
	}
	go support.RunWithRecovery(r, nil)
}
