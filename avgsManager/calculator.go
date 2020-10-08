package avgsManager

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
)

// TODO this process will receive every new value and calculate the averages as indicated in a measurement.ini
func calculator(rst chan interface{}) {

	//realTimeDefinitions := make(map[string]int)
	//referenceDefinitions := make(map[string]int)

	// load definitions of measurements from measurements.ini
	definitions, err := ini.InsensitiveLoad("measurements.ini")
	if err != nil {
		fmt.Printf("Fail to read measurements.ini file: %v", err)
		os.Exit(1)
	}

	//tick :=

	for _, def := range definitions.Section("realtime").KeyStrings() {
		println(def)
	}

	for _, def := range definitions.Section("reference").KeyStrings() {
		println(def)
	}

	os.Exit(0)

	select {
	case <-rst:
		fmt.Println("closing calculator")
		rst <- nil
	}
}
