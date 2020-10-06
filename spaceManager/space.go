package spaceManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/storage/coredbs"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"os"
	"strconv"
	"sync"
)

var once sync.Once

// TODO add shadowspacing (saving real data vs adjusted data)
//  it needs the empty space interval

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
			//if globals.DebugActive {
			//	fmt.Printf("Space %v received data \n\t%+v\n", spacename, data)
			//}
			if data.Count != 0 && spaceRegister.State {
				if _, ok := entries[data.Id]; ok {
					data.Reversed = entries[data.Id].Reversed
					if !globals.AcceptNegatives {
						delta := spaceRegister.Count
						if data.Reversed {
							delta -= (data.Count - spaceRegister.Flows[data.Id].Count)
						} else {
							delta += (data.Count - spaceRegister.Flows[data.Id].Count)
						}
						if delta < 0 {
							// count total is negative, entry needs to be adjusted
							//fmt.Printf("\n!!!!! data adjusted data:%v oldFlow:%v oldtotal:%v delta:%v reversed:%v\n",
							//	data.Count, spaceRegister.Flows[data.Id].Count, spaceRegister.Count, delta, data.Reversed)
							tmp := dataformats.EntryState{
								Id: data.Id,
								Ts: data.Ts,
								//Count:    spaceRegister.Flows[data.Id].Count - spaceRegister.Count,
								State:    data.State,
								Reversed: data.Reversed,
								Flows:    make(map[string]dataformats.Flow),
							}
							//tmp.Flows = make(map[string]dataformats.Flow)
							for key, value := range data.Flows {
								// since data was duplicated before being send, we can use a shallow copy
								tmp.Flows[key] = value
							}
							if tmp.Reversed {
								delta *= -1
							}
							tmp.Count = data.Count - delta

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
							data = tmp
						}
					}
					spaceRegister.Flows[data.Id] = data
					spaceRegister.Ts = data.Ts
					spaceRegister.Count = 0
					for _, entry := range spaceRegister.Flows {
						if entry.Reversed {
							spaceRegister.Count -= entry.Count
						} else {
							spaceRegister.Count += entry.Count
						}
					}
					if globals.DebugActive {
						fmt.Printf("Space %v registry data \n\t%+v\n", spacename, spaceRegister)
					}

					gateCount := 0
					entryCount := 0
					for key, entryFlow := range spaceRegister.Flows {
						tempGateCount := 0
						for _, gateflow := range entryFlow.Flows {
							tempGateCount += gateflow.In + gateflow.Out
						}
						//if entryFlow.Reversed {
						if entries[key].Reversed {
							entryCount -= entryFlow.Count
							gateCount -= tempGateCount
						} else {
							entryCount += entryFlow.Count
							gateCount += tempGateCount
						}
					}

					if spaceRegister.Count != entryCount || spaceRegister.Count != gateCount {
						spaceRegister.Invalid = true
						if globals.DebugActive {
							fmt.Printf("Space %v report error in data total:%v entry:%v gate:%v\n",
								spacename, spaceRegister.Count, entryCount, gateCount)
							others.PrettyPrint(spaceRegister)
							os.Exit(0)
						} else {
							mlogger.Warning(globals.SpaceManagerLog,
								mlogger.LoggerData{"entryManager.entry wroing count: " + spacename,
									"total:" + strconv.Itoa(spaceRegister.Count) + " entry:" +
										strconv.Itoa(entryCount) + " gate:" + strconv.Itoa(gateCount),
									[]int{0}, true})
						}
					} else {
						spaceRegister.Invalid = false
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
					mlogger.Warning(globals.SpaceManagerLog,
						mlogger.LoggerData{"entryManager.entry: " + spacename,
							"data from entry " + data.Id + " not in configuration",
							[]int{0}, true})
				}
			}
		}
	}

}
