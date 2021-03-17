package entryManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/spaceManager"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"time"
)

// entry receives the gate data and includes in an entry data adjusting all values based on the reversed flag
// it transmits only variations (already adjusted for the reversed flag)
func entry(entryname string, entryRegister dataformats.EntryState, in chan dataformats.FlowData, stop chan interface{},
	setReset chan bool, gates map[string]dataformats.GateState) {
	defer func() {
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
			fmt.Println("Closing entryManager.entry:", entryname)
			mlogger.Info(globals.EntryManagerLog,
				mlogger.LoggerData{"entryManager.entry: " + entryname,
					"service stopped",
					[]int{0}, true})
			stop <- nil
			break
		case data := <-in:
			if _, ok := gates[data.Name]; data.Netflow != 0 && entryRegister.State && ok {
				entryRegister.Variation = data.Netflow
				reversed := gates[data.Name].Reversed
				if reversed {
					entryRegister.Variation = -data.Netflow
				}
				// the data variation is sent, accumulation is done somewhere else
				tempRegister := dataformats.Flow{
					Id:        data.Name,
					Variation: data.Netflow,
					Reversed:  gates[data.Name].Reversed,
				}
				entryRegister.Flows[data.Name] = tempRegister
				for key := range entryRegister.Flows {
					if key != data.Name {
						tmp := entryRegister.Flows[key]
						tmp.Variation = 0
						entryRegister.Flows[key] = tmp
					}
				}

				if globals.DebugActive {
					fmt.Printf("Entry %v registry data \n\t%+v\n", entryname, entryRegister)
				}
				ts := time.Now().UnixNano()
				for _, ch := range entrySpaceChannels {
					// to avoid pointer issues we make identical deep copies instead of one
					copyState := dataformats.EntryState{
						Id:        entryRegister.Id,
						Ts:        ts,
						Variation: entryRegister.Variation,
						State:     entryRegister.State,
						Reversed:  entryRegister.Reversed,
						Flows:     make(map[string]dataformats.Flow),
					}
					for key, el := range entryRegister.Flows {
						tmp := dataformats.Flow{
							Id:        el.Id,
							Variation: el.Variation,
							Reversed:  el.Reversed,
						}
						if reversed {
							tmp.Variation = -tmp.Variation
						}
						copyState.Flows[key] = tmp
					}
					ch <- copyState
				}
			}
		}
	}

}
