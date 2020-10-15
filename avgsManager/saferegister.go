package avgsManager

import (
	"gateserver/dataformats"
)

// LatestMeasurementRegister implement a generic single input (n) single output (o)
// non-blocking register. It blocks only when it is not initialised

func LatestMeasurementRegister(tag string, in chan dataformats.SimpleSample, out chan map[string]dataformats.SimpleSample, data map[string]dataformats.SimpleSample) {
	//fmt.Println(tag, "started")

	defer func() {
		if err := recover(); err != nil {
			LatestMeasurementRegister(tag, in, out, data)
		}
	}()

	if data == nil {
		data = make(map[string]dataformats.SimpleSample)
	}
	for {
		//fmt.Println("register ->", tag, data)
		select {
		case newData := <-in:
			//fmt.Println(tag, data)
			data[newData.Qualifier] = newData
		case out <- data:
		}
	}
}
