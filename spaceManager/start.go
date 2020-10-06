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
	EntryStructure.DataChannel = make(map[string]([]chan dataformats.EntryState))
	SpaceStructure.EntryList = make(map[string]map[string]dataformats.EntryState)
	SpaceStructure.DataChannel = make(map[string]chan dataformats.EntryState)
	SpaceStructure.SetReset = make(map[string]chan bool)
	SpaceStructure.StopChannel = make(map[string]chan interface{})
	SpaceStructure.reset = false

	for _, sp := range globals.Config.Section("spaces").KeyStrings() {
		currentSpace := sp
		slot := globals.Config.Section("reset").Key(currentSpace).MustString("")
		if slot != "" {
			var start, stop time.Time
			valid := false
			period := strings.Split(slot, " ")
			if len(period) == 2 {
				if v, e := time.Parse(globals.TimeLayoutDot, strings.Trim(period[0], " ")); e == nil {
					start = v
					if v, e = time.Parse(globals.TimeLayoutDot, strings.Trim(period[1], " ")); e == nil {
						stop = v
						valid = true
					}
				}
				if valid {
					if SpaceStructure.ResetTime == nil {
						SpaceStructure.ResetTime = make(map[string][]time.Time)
					}
					SpaceStructure.ResetTime[currentSpace] = []time.Time{start, stop}
					//fmt.Println(currentSpace,start,stop)
				}
			}
		}
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
				newDataChannel := make(chan dataformats.EntryState, globals.ChannellingLength)
				for _, ce := range entries {
					ceName := strings.Trim(strings.Replace(ce, "!", "", -1), "")
					EntryStructure.SpaceList[ceName] = append(EntryStructure.SpaceList[ceName], currentSpace)
					EntryStructure.DataChannel[ceName] = append(EntryStructure.DataChannel[ceName], newDataChannel)
					if SpaceStructure.EntryList[currentSpace] == nil {
						SpaceStructure.EntryList[currentSpace] = make(map[string]dataformats.EntryState)
					}
					SpaceStructure.EntryList[currentSpace][ceName] = dataformats.EntryState{
						Id:       ceName,
						Ts:       0,
						Count:    0,
						State:    true,
						Reversed: strings.Contains(ce, "!"),
						Flows:    nil,
					}
				}

				// channels are created only if the sensor list is valid
				if SpaceStructure.EntryList[currentSpace] != nil {
					SpaceStructure.DataChannel[currentSpace] = newDataChannel
					SpaceStructure.SetReset[currentSpace] = make(chan bool, globals.ChannellingLength)
					SpaceStructure.StopChannel[currentSpace] = make(chan interface{}, 1)
					spaceRegister := dataformats.SpaceState{
						Id:    currentSpace,
						Ts:    time.Now().UnixNano(),
						Count: 0,
						Flows: nil,
						State: true,
					}
					spaceRegister.Flows = make(map[string]dataformats.EntryState)
					for entry := range SpaceStructure.EntryList[currentSpace] {
						//spaceRegister.Flows[entry] = dataformats.EntryState{Id: entry}
						spaceRegister.Flows[entry] = SpaceStructure.EntryList[currentSpace][entry]
					}
					if slot, ok := SpaceStructure.ResetTime[currentSpace]; ok {
						go space(currentSpace, spaceRegister, SpaceStructure.DataChannel[currentSpace],
							SpaceStructure.StopChannel[currentSpace], SpaceStructure.SetReset[currentSpace],
							SpaceStructure.EntryList[currentSpace], slot)
					} else {
						go space(currentSpace, spaceRegister, SpaceStructure.DataChannel[currentSpace],
							SpaceStructure.StopChannel[currentSpace], SpaceStructure.SetReset[currentSpace],
							SpaceStructure.EntryList[currentSpace], nil)
					}

				}

			} else {
				fmt.Printf("Invalid space definition %v in configuration.ini ignored\n", currentSpace)
			}
		}
	}

	EntryStructure.Unlock()
	SpaceStructure.Unlock()

	//fmt.Println(EntryStructure)
	//fmt.Println(SpaceStructure)
	//time.Sleep(3*time.Second)
	//os.Exit(0)

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
