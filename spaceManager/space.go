package spaceManager

import (
	"fmt"
	"gateserver/avgsManager"
	"gateserver/dataformats"
	"gateserver/storage/coredbs"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"os"
	"sync"
	"time"
)

var once sync.Once

// updateRegister accumulates the count at space level and passes forth variations only for the rest adjusting for reversed
func updateRegister(spaceRegister dataformats.SpaceState, data dataformats.EntryState) dataformats.SpaceState {
	//defer func() {
	//	if v := recover(); v != nil {
	//		log.Println("capture a panic:", v)
	//		os.Exit(0)
	//	}
	//}()

	// all space entries and relative gates are reset
	for key, entry := range spaceRegister.Flows {
		entry.Variation = 0
		entry.Ts = data.Ts
		for i, val := range entry.Flows {
			val.Variation = 0
			entry.Flows[i] = val
		}
		spaceRegister.Flows[key] = entry
	}

	// we set the entry/gate variation from the new data (they are received one entry at a time)
	newEntryData := dataformats.EntryState{
		Id:       data.Id,
		Ts:       data.Ts,
		State:    data.State,
		Reversed: data.Reversed,
		Flows:    make(map[string]dataformats.Flow),
	}
	if data.Reversed {
		newEntryData.Variation = -data.Variation
	} else {
		newEntryData.Variation = data.Variation
	}
	spaceRegister.Flows[data.Id] = newEntryData
	for name, val := range data.Flows {
		newGateData := dataformats.Flow{
			Id:       val.Id,
			Reversed: val.Reversed,
		}
		if data.Reversed {
			newGateData.Variation = -val.Variation
		} else {
			newGateData.Variation = val.Variation
		}
		spaceRegister.Flows[data.Id].Flows[name] = newGateData
	}
	spaceRegister.Ts = data.Ts

	// the space count is updated
	for _, entry := range spaceRegister.Flows {
		spaceRegister.Count += entry.Variation
		//if entry.Reversed {
		//	spaceRegister.Count -= entry.Variation
		//} else {
		//	spaceRegister.Count += entry.Variation
		//}
	}
	return spaceRegister
}

func space(spacename string, spaceRegister, shadowSpaceRegister dataformats.SpaceState, in chan dataformats.EntryState, stop chan interface{},
	setReset chan bool, entries map[string]dataformats.EntryState, resetSlot []time.Time) {

	// spaceRegister contains the data to be shared with the clients
	// shadowSpaceRegister is a register copy without adjustments
	once.Do(func() {
		if resetSlot != nil {
			fmt.Printf("*** INFO: Space %v has reset slot set from %v:%v to %v:%v Server Time ***\n",
				spacename, resetSlot[0].Hour(), resetSlot[0].Minute(), resetSlot[1].Hour(), resetSlot[1].Minute())
		} else {
			fmt.Printf("*** INFO: Space %v has not reset slot ***\n", spacename)
		}
		// TODO time check can be removed if a cache is used that has timed preserve of state
		if globals.SaveState {
			if state, err := diskCache.ReadState(spacename); err == nil {
				if time.Now().UnixNano() < state.Ts+int64(globals.MaxStateAge)*1000000000 {
					if state.Id == spacename {
						spaceRegister = state
					} else {
						fmt.Println("*** WARNING: Error reading state for space:", spacename, ". Will assume null state ***")
						//os.Exit(0)
					}
				} else {
					fmt.Println("*** WARNING: State is too old for space:", spacename, ". Will assume null state ***")
				}
			}
			if state, err := diskCache.ReadShadowState(spacename); err == nil {
				if time.Now().UnixNano() < state.Ts+int64(globals.MaxStateAge)*1000000000 {
					if state.Id == spacename {
						shadowSpaceRegister = state
					} else {
						fmt.Println("*** WARNING: Error reading shadow state for space:", spacename, ". Will assume null state ***")
						//os.Exit(0)
					}
				} else {
					fmt.Println("*** WARNING: Shadow state is too old for space:", spacename, ". Will assume null state ***")
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
		go space(spacename, spaceRegister, shadowSpaceRegister, in, stop, setReset, entries, resetSlot)
	}()

	if globals.DebugActive {
		fmt.Printf("Space %v has been started\n", spacename)
	}
	mlogger.Info(globals.SpaceManagerLog,
		mlogger.LoggerData{"spaceManager.entry: " + spacename,
			"service started",
			[]int{0}, true})

	resetDone := false
	avgsManager.LatestData.RLock()
	calculator := avgsManager.LatestData.Channel[spacename]
	avgsManager.LatestData.RUnlock()

	for calculator == nil {
		fmt.Printf("*** INFO: Space %v waiting for calculator to be ready ***\n", spacename)
		time.Sleep(time.Duration(globals.SettleTime*3) * time.Second)
		avgsManager.LatestData.RLock()
		calculator = avgsManager.LatestData.Channel[spacename]
		avgsManager.LatestData.RUnlock()
	}

	for {
		select {
		case spaceRegister.State = <-setReset:
			shadowSpaceRegister.State = spaceRegister.State
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
				if err := diskCache.SaveState(spaceRegister); err != nil {
					fmt.Println("Error saving state for space:", spacename)
				} else {
					fmt.Println("Successful saving state for space:", spacename)
					if err := diskCache.SaveShadowState(shadowSpaceRegister); err != nil {
						fmt.Println("Error saving shadow state for space:", spacename)
					} else {
						fmt.Println("Successful saving shadow state for space:", spacename)
					}
				}
			}
			fmt.Println("Closing spaceManager.space:", spacename)
			mlogger.Info(globals.SpaceManagerLog,
				mlogger.LoggerData{"spacxeManager.entry: " + spacename,
					"service stopped",
					[]int{0}, true})
			stop <- nil

		case data := <-in:
			if spaceRegister.State {
				// space is enabled
				// we verify if we are in a reset slot
				resetTime := resetSlot != nil
				if resetTime {
					if inTime, err := others.InClosureTime(resetSlot[0], resetSlot[1]); err != nil {
						mlogger.Warning(globals.SpaceManagerLog,
							mlogger.LoggerData{"entryManager.entry: " + spacename,
								"failed to check reset time",
								[]int{0}, true})
						continue
					} else {
						resetTime = resetTime && inTime
					}
				}
				if resetTime {
					if !resetDone {
						select {
						case calculator <- dataformats.SpaceState{
							Reset: true,
						}:
						case <-time.After(time.Duration(2*globals.SettleTime) * time.Second):
							if globals.DebugActive {
								fmt.Println("entryManager.entry:", spacename, "failed to reset flows")
								os.Exit(0)
							}
							mlogger.Warning(globals.SpaceManagerLog,
								mlogger.LoggerData{Id: "entryManager.entry: " + spacename,
									Message: "failed to reset flows",
									Data:    []int{1}, Aggregate: true})
						}

						//println("reset")
						resetDone = true
						spaceRegister.Count = 0
						spaceRegister.Ts = time.Now().UnixNano()
						for i, entry := range spaceRegister.Flows {
							entry.Variation = 0
							entry.Flows = make(map[string]dataformats.Flow)
							spaceRegister.Flows[i] = entry
						}
						go func(nd dataformats.SpaceState) {
							_ = coredbs.SaveSpaceData(nd)
						}(spaceRegister)

					}
					// the shadow register is always kept updated
					shadowSpaceRegister = updateRegister(shadowSpaceRegister, data)
					if globals.Shadowing {
						go func(nd dataformats.SpaceState) {
							_ = coredbs.SaveShadowSpaceData(nd)
						}(shadowSpaceRegister)
					}

				} else {
					resetDone = false
					if data.Variation != 0 {
						// data is significant
						// we are in a activity slot

						if _, ok := entries[data.Id]; ok {
							//fmt.Printf("%+v\n", data)
							//fmt.Printf("%+v\n\n", entries[data.Id])
							// entry sending data is in the configuration
							data.Reversed = entries[data.Id].Reversed
							timestamp := time.Now().UnixNano()

							//fmt.Printf("%+v\n", shadowSpaceRegister)
							//fmt.Printf("%+v\n", data)

							// the shadow register is updated with the received data
							shadowSpaceRegister = updateRegister(shadowSpaceRegister, data)
							shadowSpaceRegister.Ts = timestamp

							//fmt.Printf("%+v\n\n", shadowSpaceRegister)

							//continue

							// the data is updated in case it leads to a negative count if the option is enabled
							if !globals.AcceptNegatives {
								newData := data.Variation
								if data.Reversed {
									newData = -newData
								}
								delta := newData + spaceRegister.Count
								if delta < 0 {
									//println("negative check triggered")
									// the new data brings the final count below zero

									// the total count is updated according to the reversed flag in order to have final count zero
									if data.Reversed {
										data.Variation = spaceRegister.Count
									} else {
										data.Variation = -spaceRegister.Count
									}

									// the new data gate flows are updated according to the delta and the reversed flag
									entry := dataformats.EntryState{
										Id:        data.Id,
										Ts:        data.Ts,
										Variation: data.Variation,
										State:     data.State,
										Reversed:  data.Reversed,
										Flows:     make(map[string]dataformats.Flow),
									}
									for key, value := range data.Flows {
										// since data was duplicated before being sent, we can use a shallow copy
										entry.Flows[key] = value
									}

									//if entry.Reversed {
									//	delta *= -1
									//}

									// all variations are removed
									for i := range entry.Flows {
										flow := entry.Flows[i]
										flow.Variation = 0
										entry.Flows[i] = flow
									}

									delta = data.Variation
									//fmt.Println(delta)

									// the new variation is distributed among all gates in the original data
								finished:
									for delta != 0 {
										for i := range entry.Flows {
											if delta < 0 {
												flow := entry.Flows[i]
												flow.Variation -= 1
												entry.Flows[i] = flow
												delta += 1
											} else if delta > 0 {
												flow := entry.Flows[i]
												flow.Variation += 1
												entry.Flows[i] = flow
												delta -= 1
											} else {
												break finished
											}
										}
									}

									data = entry
								}
							}

							//fmt.Printf("%+v\n", spaceRegister)
							//fmt.Printf("%+v\n", data)

							// register is updated with an inspected received data
							spaceRegister = updateRegister(spaceRegister, data)
							// space gets its own timestamp
							spaceRegister.Ts = timestamp

							//fmt.Printf("%+v\n\n", spaceRegister)
							//continue

							if globals.DebugActive {
								fmt.Printf("Space %v registry data \n\t%+v\n", spacename, spaceRegister)
							}

							//fmt.Println(spacename,"sending data", spaceRegister)
							//fmt.Println(spaceRegister)
							//fmt.Println(shadowSpaceRegister)
							//continue

							go func(nd dataformats.SpaceState) {
								_ = coredbs.SaveSpaceData(nd)
							}(spaceRegister)

							// we give it little time to transmit the data, it too late data is thrown away
							select {
							case calculator <- spaceRegister:
							case <-time.After(time.Duration(globals.SettleTime) * time.Second):
								if globals.DebugActive {
									fmt.Println("entryManager.entry:", spacename, "data to calculator discarded due to late answer")
									os.Exit(0)
								}
								mlogger.Warning(globals.SpaceManagerLog,
									mlogger.LoggerData{Id: "entryManager.entry: " + spacename,
										Message: "data to calculator discarded due to late answer",
										Data:    []int{1}, Aggregate: true})
							}

							if globals.Shadowing {
								go func(nd dataformats.SpaceState) {
									_ = coredbs.SaveShadowSpaceData(nd)
								}(spaceRegister)
							}
						} else {
							mlogger.Warning(globals.SpaceManagerLog,
								mlogger.LoggerData{Id: "entryManager.entry: " + spacename,
									Message: "data from entry " + data.Id + " not in configuration",
									Data:    []int{0}, Aggregate: true})
						}
					}
				}
			}
		}
	}
}
