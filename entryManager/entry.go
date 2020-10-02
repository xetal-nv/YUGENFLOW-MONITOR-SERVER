package entryManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
)

func entry(entryname string, entryRegister entryData, in chan dataformats.FlowData, stop chan interface{}, resetEntry chan interface{}, gates map[string]dataformats.GateDefinition) {

	// TODO everything, must include dumping data as well?
	// TODO we need to be able to read the last state from somewhere (is enabled, reset otherwise)

	defer func() {
		if e := recover(); e != nil {
			mlogger.Recovered(globals.GateManagerLog,
				mlogger.LoggerData{"entryManager.entry: " + entryname,
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		}
		go entry(entryname, entryRegister, in, stop, resetEntry, gates)
	}()

	fmt.Printf("Entry %v has been started\n", entryname)

	for {
		select {
		case <-resetEntry:
			fmt.Println("Resetting entryManager.entry:", entryname)
		case <-stop:
			// TODO we need to be able to save the last state somewhere
			fmt.Println("Closing entryManager.entry:", entryname)
			stop <- nil
		case data := <-in:
			if data.Netflow != 0 {
				if _, ok := gates[data.Name]; ok {
					if gates[data.Name].Reversed {
						data.Netflow *= -1
					}
				}
				entryRegister.count += data.Netflow
				tempRegister := gateflow{
					name: data.Name,
					in:   entryRegister.flows[data.Name].in,
					out:  entryRegister.flows[data.Name].out,
				}
				if data.Netflow < 0 {
					tempRegister.out += data.Netflow
				} else {
					tempRegister.in += data.Netflow
				}
				entryRegister.flows[data.Name] = tempRegister
				//fmt.Println(entryRegister.flows[data.Name])
				// TODO send to database and space
				fmt.Printf("Entry %v registry data \n\t%+v\n", entryname, entryRegister)
			}
		}
	}

}
