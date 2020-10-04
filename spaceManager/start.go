package spaceManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"strings"
	"sync"
	"time"
)

func Start(sd chan bool) {
	var err error

	if globals.SpaceManagerLog, err = mlogger.DeclareLog("yugenflow_spaceManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_spaceManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.SpaceManagerLog, 40, 30, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.SpaceManagerLog,
		mlogger.LoggerData{"spaceManager.Start",
			"service started",
			[]int{0}, true})

	EntryStructure.Lock()
	SpaceStructure.Lock()
	EntryStructure.SpaceList = make(map[string][]string)
	EntryStructure.DataChannel = make(map[string]([]chan dataformats.Entrydata))
	SpaceStructure.EntryList = make(map[string]map[string]dataformats.GateDefinition)
	SpaceStructure.DataChannel = make(map[string]chan dataformats.Entrydata)
	SpaceStructure.SetReset = make(map[string]chan bool)
	SpaceStructure.StopChannel = make(map[string]chan interface{})

	for _, sp := range globals.Config.Section("spaces").KeyStrings() {
		currentSpace := sp
		if _, ok := SpaceStructure.EntryList[currentSpace]; ok {
			fmt.Println("Duplicated entry %v in configuration.ini ignored\n", currentSpace)
		} else {
			spaceDef := globals.Config.Section("spaces").Key(currentSpace).MustString("")
			if spaceDef != "" {
				entries := strings.Split(spaceDef, " ")
				// semantics check of the entry definitions (reject gate and !gate in the same gate)
				for i, s := range entries {
					entries[i] = strings.Trim(strings.Replace(s, "!", "", -1), "")
				}
				illegal := false
				for _, s := range entries {
					illegal = strings.Contains(" "+spaceDef, " "+s) && strings.Contains(spaceDef, "!"+s)
					if illegal {
						break
					}
				}

				if illegal {
					fmt.Printf("Invalid space definition \"%v : %v\" in configuration.ini ignored.\n",
						currentSpace, spaceDef)
					continue
				}

				entries = strings.Split(spaceDef, " ")
				newDataChannel := make(chan dataformats.Entrydata, globals.ChannellingLength)
				for _, ce := range entries {
					ceName := strings.Trim(strings.Replace(ce, "!", "", -1), "")
					EntryStructure.SpaceList[ceName] = append(EntryStructure.SpaceList[ceName], currentSpace)
					EntryStructure.DataChannel[ceName] = append(EntryStructure.DataChannel[ceName], newDataChannel)
					if SpaceStructure.EntryList[currentSpace] == nil {
						SpaceStructure.EntryList[currentSpace] = make(map[string]dataformats.GateDefinition)
					}
					// TODO need to change type to also add data on gates and entries
					SpaceStructure.EntryList[currentSpace][ceName] = dataformats.GateDefinition{
						Id:        ceName,
						Reversed:  strings.Contains(ce, "!"),
						Suspected: 0,
						Disabled:  false,
					}
				}

				// channels are created only if the sensor list is valid
				if SpaceStructure.EntryList[currentSpace] != nil {
					SpaceStructure.DataChannel[currentSpace] = newDataChannel
					SpaceStructure.SetReset[currentSpace] = make(chan bool, globals.ChannellingLength)
					SpaceStructure.StopChannel[currentSpace] = make(chan interface{}, 1)
					entryData := dataformats.Entrydata{
						Id:    currentSpace,
						Ts:    time.Now().UnixNano(),
						Count: 0,
						Flows: nil,
						State: true,
					}
					entryData.Flows = make(map[string]dataformats.Flow)
					for gate := range SpaceStructure.EntryList[currentSpace] {
						entryData.Flows[gate] = dataformats.Flow{Id: gate}
					}
					//go entry(currentSpace, entryData, SpaceStructure.DataChannel[currentSpace], SpaceStructure.StopChannel[currentSpace],
					//	SpaceStructure.SetReset[currentSpace], SpaceStructure.GateList[currentSpace])
				}

			} else {
				fmt.Printf("Invalid space definition %v in configuration.ini ignored\n", currentSpace)
			}
		}
	}

	EntryStructure.Unlock()
	SpaceStructure.Unlock()

	fmt.Println(EntryStructure)
	fmt.Println(SpaceStructure)
	os.Exit(0)

	// setting up closure and shutdown
	<-sd
	fmt.Println("Closing spaceManager")
	var wg sync.WaitGroup
	SpaceStructure.Lock()
	for _, ch := range SpaceStructure.StopChannel {
		wg.Add(1)
		go func(ch chan interface{}) {
			ch <- nil
			select {
			case <-ch:
			case <-time.After(time.Duration(globals.SettleTime) * time.Second):
			}
			wg.Done()
		}(ch)
	}
	SpaceStructure.Unlock()
	wg.Wait()
	mlogger.Info(globals.SpaceManagerLog,
		mlogger.LoggerData{"spaceManager.Start",
			"service stopped",
			[]int{0}, true})
	time.Sleep(time.Duration(globals.SettleTime) * time.Second)
	sd <- true

	//for _, current := range globals.Config.Section("spaces").KeyStrings() {
	//	fmt.Println(current, globals.Config.Section("spaces").Key(current))
	//}

}
