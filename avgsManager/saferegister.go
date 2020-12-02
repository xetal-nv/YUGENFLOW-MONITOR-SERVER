package avgsManager

import (
	"gateserver/dataformats"
)

// LatestMeasurementRegister implement a generic single input (n) single output (o)
// non-blocking register. It blocks only when it is not initialised

func LatestMeasurementRegister(tag string, in chan dataformats.MeasurementSample, out chan map[string]dataformats.MeasurementSample, data map[string]dataformats.MeasurementSample) {
	//fmt.Println(tag, "started")

	defer func() {
		if err := recover(); err != nil {
			LatestMeasurementRegister(tag, in, out, data)
		}
	}()

	if data == nil {
		data = make(map[string]dataformats.MeasurementSample)
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

// LatestMeasurementRegisterActuals implement a generic single input (n) single output (o)
// non-blocking register. It blocks only when it is not initialised

func LatestMeasurementRegisterActuals(tag string, in, out chan dataformats.MeasurementSampleWithFlows) {
	//fmt.Println(tag, "started")

	defer func() {
		if err := recover(); err != nil {
			print(err)
			LatestMeasurementRegisterActuals(tag, in, out)
		}
	}()

	data := dataformats.MeasurementSampleWithFlows{}

	for {
		//fmt.Println("register ->", tag, data)
		select {
		case data = <-in:
			//fmt.Printf("%v gets %+v\n",tag, data)
		case out <- data:
			//fmt.Printf("%v sends %+v\n",tag, data)
		}
	}
}
