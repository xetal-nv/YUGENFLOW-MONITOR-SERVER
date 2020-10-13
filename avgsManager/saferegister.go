package avgsManager

import (
	"fmt"
	"gateserver/dataformats"
)

// LatestMeasurementRegister implement a generic single input (n) single output (o)
// non-blocking register. It blocks only when it is not initialised

// TODO needs to differentiate values and store them in a map !!!
func LatestMeasurementRegister(tag string, in, out chan dataformats.SimpleSample, data dataformats.SimpleSample) {
	//fmt.Println(tag, "started")
	for {
		fmt.Println("register ->", tag, data)
		select {
		case data = <-in:
			//fmt.Println(tag, data)
		case out <- data:
		}
	}
}
