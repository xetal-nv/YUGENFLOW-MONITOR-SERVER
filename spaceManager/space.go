package spaceManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/storage/coredbs"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"os"
	"sync"
	"time"
)

var once sync.Once

func updateRegister(spaceRegister dataformats.SpaceState, data dataformats.EntryState) dataformats.SpaceState {
	spaceRegister.Flows[data.Id] = data
	for key, entry := range spaceRegister.Flows {
		if key != data.Id {
			entry.Count = 0
			for i, val := range entry.Flows {
				val.Variation = 0
				entry.Flows[i] = val
			}
			//entry.Flows = make(map[string]dataformats.Flow)
			spaceRegister.Flows[key] = entry
		} else {
			entry.Count = data.Count
			entry.Reversed = data.Reversed
			entry.State = data.State
			entry.Ts = data.Ts
			for i, val := range entry.Flows {
				val.Variation = data.Flows[i].Variation
				entry.Flows[i] = val
			}
		}
	}
	spaceRegister.Ts = data.Ts
	//spaceRegister.Count = 0
	for _, entry := range spaceRegister.Flows {
		if entry.Reversed {
			spaceRegister.Count -= entry.Count
		} else {
			spaceRegister.Count += entry.Count
		}
	}
	return spaceRegister
}

func space(spacename string, spaceRegister, shadowSpaceRegister dataformats.SpaceState, in chan dataformats.EntryState, stop chan interface{},
	setReset chan bool, entries map[string]dataformats.EntryState, resetSlot []time.Time) {

	// spaceRegister contains the data to be shared with the clients
	// shadowSpaceRegister is a register copy without reset form debugging. It might overflow
	once.Do(func() {
		if resetSlot != nil {
			fmt.Printf("*** Space %v has reset slot set from %v:%v to %v:%v Server Time ***\n",
				spacename, resetSlot[0].Hour(), resetSlot[0].Minute(), resetSlot[1].Hour(), resetSlot[1].Minute())
		} else {
			fmt.Printf("*** Space %v has not reset slot ***\n", spacename)
		}
		if globals.SaveState {
			if state, err := coredbs.LoadSpaceState(spacename); err == nil {
				if state.Id == spacename {
					spaceRegister = state
				} else {
					fmt.Println("Error reading state for space:", spacename)
					os.Exit(0)
				}
			}
			if state, err := coredbs.LoadSpaceShadowState(spacename); err == nil {
				if state.Id == spacename {
					shadowSpaceRegister = state
				} else {
					fmt.Println("Error reading shadow state for space:", spacename)
					os.Exit(0)
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
		mlogger.LoggerData{"entryManager.entry: " + spacename,
			"service started",
			[]int{0}, true})

	resetDone := false

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
				if err := coredbs.SaveSpaceState(spacename, spaceRegister); err != nil {
					fmt.Println("Error saving state for space:", spacename)
				} else {
					fmt.Println("Successful saving state for space:", spacename)
					if err := coredbs.SaveSpaceShadowState(spacename, shadowSpaceRegister); err != nil {
						fmt.Println("Error saving shadow state for space:", spacename)
					} else {
						fmt.Println("Successful saving shadow state for space:", spacename)
					}
				}
			}
			fmt.Println("Closing spaceManager.space:", spacename)
			mlogger.Info(globals.SpaceManagerLog,
				mlogger.LoggerData{"entryManager.entry: " + spacename,
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
						println("reset")
						resetDone = true
						spaceRegister.Count = 0
						spaceRegister.Ts = time.Now().UnixNano()
						for i, entry := range spaceRegister.Flows {
							entry.Count = 0
							entry.Flows = make(map[string]dataformats.Flow)
							spaceRegister.Flows[i] = entry
						}
						go func(nd dataformats.SpaceState) {
							coredbs.SaveSpaceData(nd)
						}(spaceRegister)

					}
					// the shadow register is always kept updated
					shadowSpaceRegister = updateRegister(shadowSpaceRegister, data)
					if globals.Shadowing {
						go func(nd dataformats.SpaceState) {
							coredbs.SaveShadowSpaceData(nd)
						}(shadowSpaceRegister)
					}

				} else {
					resetDone = false
					if data.Count != 0 {
						// data is significant
						// we are in a activity slot
						if _, ok := entries[data.Id]; ok {
							// entry sending data is in the configuration
							data.Reversed = entries[data.Id].Reversed

							// the shadow register is updated with the received data
							shadowSpaceRegister = updateRegister(shadowSpaceRegister, data)

							// the data is updated in case it leads to a negative count if the option is enabled
							if !globals.AcceptNegatives {
								newData := data.Count
								if data.Reversed {
									newData = -newData
								}
								delta := newData + spaceRegister.Count
								if delta < 0 {
									// the new data brings the final count below zero

									// the total count is updated according to the reversed flag
									if data.Reversed {
										data.Count = spaceRegister.Count
									} else {
										data.Count = -spaceRegister.Count
									}

									// the gate flows are updated according to the delta and the reversed flag
									entry := dataformats.EntryState{
										Id:       data.Id,
										Ts:       data.Ts,
										Count:    data.Count,
										State:    data.State,
										Reversed: data.Reversed,
										Flows:    make(map[string]dataformats.Flow),
									}
									for key, value := range data.Flows {
										// since data was duplicated before being sent, we can use a shallow copy
										entry.Flows[key] = value
									}
									if entry.Reversed {
										delta *= -1
									}

									// the error is distributed among all flows
								finished:
									for delta != 0 {
										for i := range entry.Flows {
											if delta < 0 {
												flow := entry.Flows[i]
												flow.Variation += 1
												entry.Flows[i] = flow
												delta += 1
											} else if delta > 0 {
												flow := entry.Flows[i]
												flow.Variation -= 1
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

							// register is updated with an inspected received data
							spaceRegister = updateRegister(spaceRegister, data)

							if globals.DebugActive {
								fmt.Printf("Space %v registry data \n\t%+v\n", spacename, spaceRegister)
							}

							//fmt.Println()
							//fmt.Println(spaceRegister)
							//fmt.Println(shadowSpaceRegister)

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
	}
}
