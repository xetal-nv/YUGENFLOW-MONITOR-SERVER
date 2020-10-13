package avgsManager

import (
	"fmt"
	"gateserver/dataformats"
)

// SafeRegister implement a generic single input (n) single output (o)
// non-blocking register. It blocks only when it is not initialised

func SafeRegister(tag string, in, out chan dataformats.SimpleSample, data dataformats.SimpleSample) {
	//fmt.Println(tag, "started")
	for {
		fmt.Println(tag, data)
		select {
		case data = <-in:
			//fmt.Println(tag, data)
		case out <- data:
		}
	}
}
