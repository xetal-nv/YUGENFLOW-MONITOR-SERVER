package entryManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/spaceManager"
	"gateserver/storage/coredbs"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"sync"
	"time"
)

var once sync.Once

func entry(entryname string, entryRegister dataformats.EntryState, in chan dataformats.FlowData, stop chan interface{},
	setReset chan bool, gates map[string]dataformats.GateState) {

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
		println("a")
		if e := recover(); e != nil {
			mlogger.Recovered(globals.EntryManagerLog,
				mlogger.LoggerData{"entryManager.entry: " + entryname,
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
			go entry(entryname, entryRegister, in, stop, setReset, gates)
		}
	}()

	tries := 5
	spaceManager.EntryStructure.RLock()
	entrySpaceChannels, ok := spaceManager.EntryStructure.DataChannel[entryname]
	spaceManager.EntryStructure.RUnlock()
	for !ok {
		if tries == 0 {
			fmt.Printf("Entry %v has failed to start or is not used\n", entryname)
			os.Exit(0)
		} else {
			tries -= 1
		}
		time.Sleep(time.Duration(globals.SettleTime) * time.Second)
		spaceManager.EntryStructure.RLock()
		entrySpaceChannels, ok = spaceManager.EntryStructure.DataChannel[entryname]
		spaceManager.EntryStructure.RUnlock()
	}

	if globals.DebugActive {
		fmt.Printf("Entry %v has been started\n", entryname)
	}
	mlogger.Info(globals.EntryManagerLog,
		mlogger.LoggerData{"entryManager.entry: " + entryname,
			"service started",
			[]int{0}, true})

	for {
		select {
		case entryRegister.State = <-setReset:
			if globals.DebugActive {
				fmt.Printf("State of entry %v set to %v\n", entryname, entryRegister.State)
			}
			setReset <- entryRegister.State
			if entryRegister.State {
				mlogger.Info(globals.EntryManagerLog,
					mlogger.LoggerData{"entryManager.entry: " + entryname,
						"state set to true",
						[]int{0}, true})
			} else {
				mlogger.Info(globals.EntryManagerLog,
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
			mlogger.Info(globals.EntryManagerLog,
				mlogger.LoggerData{"entryManager.entry: " + entryname,
					"service stopped",
					[]int{0}, true})
			stop <- nil
			break
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
				//test := 0
				//for _, el := range entryRegister.Flows {
				//	test += el.Out + el.In
				//}
				//if test != entryRegister.Count {
				//	fmt.Println(entryRegister)
				//	//os.Exit(0)
				//} else {
				//	fmt.Println(entryRegister.Count, data.Netflow, entryRegister.Flows[data.Name])
				//}
				//fmt.Println(entryRegister.flows[data.Id])
				if saveToDB {
					go func(nd dataformats.EntryState) {
						coredbs.SaveEntryData(nd)
					}(entryRegister)
				}
				if globals.DebugActive {
					fmt.Printf("Entry %v registry data \n\t%+v\n", entryname, entryRegister)
				}
				ts := time.Now().UnixNano()
				for _, ch := range entrySpaceChannels {
					// to avoid pointer issues we make a deep copy of the register to send
					copyState := dataformats.EntryState{
						Id:       entryRegister.Id,
						Ts:       ts,
						Count:    entryRegister.Count,
						State:    entryRegister.State,
						Reversed: entryRegister.Reversed,
						Flows:    make(map[string]dataformats.Flow),
					}
					for key, el := range entryRegister.Flows {
						tmp := dataformats.Flow{
							Id:  el.Id,
							In:  el.In,
							Out: el.Out,
						}
						copyState.Flows[key] = tmp
					}
					ch <- copyState
				}
			}
		}
	}

}
