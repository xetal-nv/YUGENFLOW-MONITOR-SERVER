package entryManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"strings"
	"sync"
	"time"
	"xetal.ddns.net/utils/recovery"
)

func Start(sd chan bool) {
	var err error

	if globals.EntryManagerLog, err = mlogger.DeclareLog("yugenflow_entryManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_entryManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.EntryManagerLog, 40, 30, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.EntryManagerLog,
		mlogger.LoggerData{"entryManager.Start",
			"service started",
			[]int{0}, true})

	GateStructure.Lock()
	EntryStructure.Lock()
	GateStructure.EntryList = make(map[string][]string)
	GateStructure.DataChannel = make(map[string]([]chan dataformats.FlowData))
	EntryStructure.GateList = make(map[string]map[string]dataformats.GateDefinition)
	EntryStructure.DataChannel = make(map[string]chan dataformats.FlowData)
	EntryStructure.ConfigurationReset = make(map[string]chan interface{})
	EntryStructure.StopChannel = make(map[string]chan interface{})

	for _, en := range globals.Config.Section("entries").KeyStrings() {
		currentEntry := en
		if _, ok := EntryStructure.GateList[currentEntry]; ok {
			fmt.Println("Duplicated entry %v in configuration.ini ignored\n", currentEntry)
		} else {
			entryDef := globals.Config.Section("entries").Key(currentEntry).MustString("")
			if entryDef != "" {
				gates := strings.Split(entryDef, " ")
				// semantics check of the entry definitions (reject gate and !gate in the same gate)
				for i, s := range gates {
					gates[i] = strings.Trim(strings.Replace(s, "!", "", -1), "")
				}
				illegal := false
				for _, s := range gates {
					illegal = strings.Contains(" "+entryDef, " "+s) && strings.Contains(entryDef, "!"+s)
					if illegal {
						break
					}
				}

				if illegal {
					fmt.Printf("Invalid entry definition \"%v : %v\" in configuration.ini ignored.\n",
						currentEntry, entryDef)
					continue
				}

				gates = strings.Split(entryDef, " ")
				newDataChannel := make(chan dataformats.FlowData, globals.ChannellingLength)
				for _, cg := range gates {
					cgName := strings.Trim(strings.Replace(cg, "!", "", -1), "")
					GateStructure.EntryList[cgName] = append(GateStructure.EntryList[cgName], currentEntry)
					GateStructure.DataChannel[cgName] = append(GateStructure.DataChannel[cgName], newDataChannel)
					if EntryStructure.GateList[currentEntry] == nil {
						EntryStructure.GateList[currentEntry] = make(map[string]dataformats.GateDefinition)
					}
					EntryStructure.GateList[currentEntry][cgName] = dataformats.GateDefinition{
						Id:        cgName,
						Reversed:  strings.Contains(cg, "!"),
						Suspected: 0,
						Disabled:  false,
					}
				}

				// channels are created only if the sensor list is valid
				if EntryStructure.GateList[currentEntry] != nil {
					EntryStructure.DataChannel[currentEntry] = newDataChannel
					EntryStructure.ConfigurationReset[currentEntry] = make(chan interface{}, 1)
					EntryStructure.StopChannel[currentEntry] = make(chan interface{}, 1)
					go recovery.RunWith(
						func() {
							entry(currentEntry, EntryStructure.DataChannel[currentEntry], EntryStructure.StopChannel[currentEntry],
								EntryStructure.ConfigurationReset[currentEntry], EntryStructure.GateList[currentEntry])
						},
						func() {
							mlogger.Recovered(globals.GateManagerLog,
								mlogger.LoggerData{"entryManager.entry: " + currentEntry,
									"service terminated and recovered unexpectedly",
									[]int{1}, true})
						})
				}

			} else {
				fmt.Printf("Invalid entry definition %v in configuration.ini ignored\n", currentEntry)
			}
		}
	}

	GateStructure.Unlock()
	EntryStructure.Unlock()

	// setting up closure and shutdown
	<-sd
	fmt.Println("Closing entryManager")
	var wg sync.WaitGroup
	EntryStructure.Lock()
	for _, ch := range EntryStructure.StopChannel {
		wg.Add(1)
		go func(ch chan interface{}) {
			ch <- nil
			select {
			case <-ch:
			case <-time.After(time.Duration(globals.ShutdownTime) * time.Second):
			}
			wg.Done()
		}(ch)
	}
	EntryStructure.Unlock()
	wg.Wait()
	mlogger.Info(globals.EntryManagerLog,
		mlogger.LoggerData{"entryManager.Start",
			"service stopped",
			[]int{0}, true})
	time.Sleep(time.Duration(globals.ShutdownTime) * time.Second)
	sd <- true

	//for _, current := range globals.Config.Section("spaces").KeyStrings() {
	//	fmt.Println(current, globals.Config.Section("spaces").Key(current))
	//}

}
