package entryManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/storage/coredbs"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"sync"
)

var once sync.Once

func entry(entryname string, entryRegister dataformats.Entrydata, in chan dataformats.FlowData, stop chan interface{},
	setReset chan bool, gates map[string]dataformats.GateDefinition) {

	once.Do(func() {
		if globals.SaveState {
			if state, err := coredbs.LoadEntryState(entryname); err == nil {
				if state.Id == entryname {
					entryRegister = state
				} else {
					fmt.Println("Error reading state for entry:", entryname)
				}
			}
		}
	})

	defer func() {
		if e := recover(); e != nil {
			mlogger.Recovered(globals.GateManagerLog,
				mlogger.LoggerData{"entryManager.entry: " + entryname,
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		}
		go entry(entryname, entryRegister, in, stop, setReset, gates)
	}()

	fmt.Printf("Entry %v has been started\n", entryname)

	for {
		select {
		case entryRegister.State = <-setReset:
			fmt.Printf("State of entry %v set to %v\n", entryname, entryRegister.State)
			setReset <- entryRegister.State
			if entryRegister.State {
				mlogger.Info(globals.GateManagerLog,
					mlogger.LoggerData{"entryManager.entry: " + entryname,
						"state set to true",
						[]int{0}, true})
			} else {
				mlogger.Info(globals.GateManagerLog,
					mlogger.LoggerData{"entryManager.entry: " + entryname,
						"state set to false",
						[]int{0}, true})
			}
		case <-stop:
			if globals.SaveState {
				if err := coredbs.SaveEntryState(entryname, entryRegister); err != nil {
					fmt.Println("Error saving state for entry:", entryname)
				} else {
					fmt.Println("Successful saving state for entry:", entryname)
				}
			}
			fmt.Println("Closing entryManager.entry:", entryname)
			stop <- nil
		case data := <-in:
			if data.Netflow != 0 && entryRegister.State {
				if _, ok := gates[data.Name]; ok {
					if gates[data.Name].Reversed {
						data.Netflow *= -1
					}
				}
				entryRegister.Count += data.Netflow
				tempRegister := dataformats.Flow{
					Id:  data.Name,
					In:  entryRegister.Flows[data.Name].In,
					Out: entryRegister.Flows[data.Name].Out,
				}
				if data.Netflow < 0 {
					tempRegister.Out += data.Netflow
				} else {
					tempRegister.In += data.Netflow
				}
				entryRegister.Flows[data.Name] = tempRegister
				//fmt.Println(entryRegister.flows[data.Id])
				if saveToDB {
					go func(nd dataformats.Entrydata) {
						coredbs.SaveEntryData(nd)
					}(entryRegister)
				}
				// TODO send to space
				fmt.Printf("Entry %v registry data \n\t%+v\n", entryname, entryRegister)
			}
		}
	}

}
