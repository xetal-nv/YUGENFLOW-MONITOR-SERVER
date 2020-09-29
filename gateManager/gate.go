package gateManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
)

func gate(in chan dataformats.FlowData, stop chan interface{}, gateName string, sensors map[int]dataformats.SensorDefinition) {
	// TODO everything
	if globals.DebugActive {
		fmt.Printf("Gate %v has been started\n", gateName)
	}
	//fmt.Println(in, stop, gateName, sensors)
	for {
		select {
		case data := <-in:
			//fmt.Println(sensors[data.Id].Reversed, data.Netflow)
			if sensors[data.Id].Reversed {
				data.Netflow *= -1
			}
			fmt.Printf(" ===>>> Gate %v received: %+v\n", gateName, data)
		case <-stop:
			if globals.DebugActive {
				fmt.Printf("Gate %v has been stopped\n", gateName)
			}
			stop <- nil
		}
	}
}
