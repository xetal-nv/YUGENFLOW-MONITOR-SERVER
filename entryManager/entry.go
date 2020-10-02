package entryManager

import (
	"fmt"
	"gateserver/dataformats"
)

func entry(entryname string, in chan dataformats.FlowData, stop chan interface{},
	resetEntry chan interface{}, gates map[string]dataformats.GateDefinition) {

	// TODO everything, must include dumping data as well?

	fmt.Printf("Entry %v has been started\n", entryname)

	for {
		select {
		case <-resetEntry:
			fmt.Println("Resetting entryManager.entry:", entryname)
		case <-stop:
			fmt.Println("Closing entryManager.entry:", entryname)
			stop <- nil
		case data := <-in:
			if _, ok := gates[data.Name]; ok {
				if gates[data.Name].Reversed {
					data.Netflow *= -1
				}
			}
			fmt.Printf("Entry %v received data %+v\n", entryname, data)
		}
	}

}
