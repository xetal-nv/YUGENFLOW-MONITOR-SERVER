package spaceManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/storage/coredbs"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"sync"
)

var once sync.Once

func space(spacename string, spaceRegister dataformats.SpaceState, in chan dataformats.EntryState, stop chan interface{},
	setReset chan bool, entries map[string]dataformats.EntryState) {

	once.Do(func() {
		if globals.SaveState {
			if state, err := coredbs.LoadSpaceState(spacename); err == nil {
				if state.Id == spacename {
					spaceRegister = state
				} else {
					fmt.Println("Error reading state for space:", spacename)
				}
			}
		}
	})

	defer func() {
		if e := recover(); e != nil {
			mlogger.Recovered(globals.SpaceManagerLog,
				mlogger.LoggerData{"spaceManager.space: " + spacename,
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		}
		go space(spacename, spaceRegister, in, stop, setReset, entries)
	}()

	if globals.DebugActive {
		fmt.Printf("Space %v has been started\n", spacename)
	}
	mlogger.Info(globals.SpaceManagerLog,
		mlogger.LoggerData{"entryManager.entry: " + spacename,
			"service started",
			[]int{0}, true})
	//fmt.Println("register:", spaceRegister)
	//fmt.Println("entries:", entries)
	//
	for {
		select {
		case spaceRegister.State = <-setReset:
			if globals.DebugActive {
				fmt.Printf("State of space %v set to %v\n", spacename, spaceRegister.State)
			}
			setReset <- spaceRegister.State
			if spaceRegister.State {
				mlogger.Info(globals.SpaceManagerLog,
					mlogger.LoggerData{"spaceManager.space: " + spacename,
						"state set to true",
						[]int{0}, true})
			} else {
				mlogger.Info(globals.SpaceManagerLog,
					mlogger.LoggerData{"spaceManager.space: " + spacename,
						"state set to false",
						[]int{0}, true})
			}
		case <-stop:
			if globals.SaveState {
				if err := coredbs.SaveSpaceState(spacename, spaceRegister); err != nil {
					fmt.Println("Error saving state for space:", spacename)
				} else {
					fmt.Println("Successful saving state for space:", spacename)
				}
			}
			fmt.Println("Closing spaceManager.space:", spacename)
			mlogger.Info(globals.SpaceManagerLog,
				mlogger.LoggerData{"entryManager.entry: " + spacename,
					"service stopped",
					[]int{0}, true})
			stop <- nil
		case data := <-in:
			//fmt.Printf("Space %v received data \n\t%+v\n", spacename, data)
			if data.Count != 0 && spaceRegister.State {
				if _, ok := entries[data.Id]; ok {
					if entries[data.Id].Reversed {
						data.Count *= -1
					}
					spaceRegister.Flows[data.Id] = data
				}
				spaceRegister.Count = 0
				for _, entry := range spaceRegister.Flows {
					spaceRegister.Count += entry.Count
				}
				if globals.DebugActive {
					fmt.Printf("Space %v registry data \n\t%+v\n", spacename, spaceRegister)
				}
				go func(nd dataformats.SpaceState) {
					coredbs.SaveSpaceData(nd)
				}(spaceRegister)
			}
		}
	}

}
