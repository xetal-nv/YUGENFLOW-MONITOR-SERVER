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

// TODO add shadowspacing (saving real data vs adjusted data)
//  it needs two options one to enable it and one for the empty space interval

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
			fmt.Println(e)
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
					// TODO adjust for the case that flows are discordant also !!!
					//  not working
					if !globals.AcceptNegatives {
						if delta := (data.Count - spaceRegister.Flows[data.Id].Count) + spaceRegister.Count; delta < 0 {
							fmt.Println("data adjusted", data.Count, spaceRegister.Flows[data.Id].Count, spaceRegister.Count, delta)
							tmp := dataformats.EntryState{
								Id:       data.Id,
								Ts:       data.Ts,
								Count:    data.Count - delta,
								State:    data.State,
								Reversed: data.Reversed,
								Flows:    nil,
							}
							tmp.Flows = make(map[string]dataformats.Flow)
							for key, value := range data.Flows {
								tmp.Flows[key] = value
							}
							// TODO adjust flows using also spaceRegister.Flows[data.Id] flows
							if tmp.Reversed {
								delta = -delta
							}
						finished:
							for delta != 0 {
								for i := range tmp.Flows {
									if delta < 0 {
										flow := tmp.Flows[i]
										flow.In += 1
										tmp.Flows[i] = flow
										delta += 1
									} else if delta > 0 {
										flow := tmp.Flows[i]
										flow.Out -= 1
										tmp.Flows[i] = flow
										delta -= 1
									} else {
										break finished
									}
								}
							}
							//tmp.Flows
							//fmt.Println("\t", data)
							data = tmp
							//fmt.Println("\t", data)
						}
					}
					spaceRegister.Flows[data.Id] = data
					spaceRegister.Count = 0
					for _, entry := range spaceRegister.Flows {
						spaceRegister.Count += entry.Count
					}
					//fmt.Println("\t", spaceRegister.Count)
					//if spaceRegister.Count < 0 {
					//	os.Exit(0)
					//}
					//for _, entry := range spaceRegister.Flows {
					//	fmt.Println(entry.Count)
					//}
					if globals.DebugActive {
						fmt.Printf("Space %v registry data \n\t%+v\n", spacename, spaceRegister)
					}
					go func(nd dataformats.SpaceState) {
						coredbs.SaveSpaceData(nd)
					}(spaceRegister)
					if globals.Shadowing {
						go func(nd dataformats.SpaceState) {
							coredbs.SaveShadowSpaceData(nd)
						}(spaceRegister)
					}
				} else {
					// report issue?
				}
			}
		}
	}

}
