package spaces

import (
	"fmt"
	"gateserver/support"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// it implements the counters, both the current one as well as the analysis averages.
// the sampler threads for a given space are started in a recursive manner
// The algorithm is built on the ordered arrival of samples that is preserved in a slice.
// It means that x[i] is newer than x[i-1] and older than x[i+1]
// prevStage channels are used for data flow
// the sync channels are used to control the operation period and its synchronisation
// NOTE: for samples we take the time weighted averages, for entries the total only on the analysis period
func sampler(spacename string, prevStageChan, nextStageChan chan spaceEntries, syncPrevious, syncNext chan bool, avgID int, once sync.Once, tn, ntn int) {
	// set-up the next analysis stage and the communication channel
	once.Do(func() {
		if avgID < (len(avgAnalysis) - 1) {
			nextStageChan = make(chan spaceEntries, bufsize)
			syncNext = make(chan bool)
			go sampler(spacename, nextStageChan, nil, syncNext, nil, avgID+1, sync.Once{}, 0, 0)
		}
	})

	stats := []int{tn, ntn}
	//statsb := []int{0}
	samplerName := avgAnalysis[avgID].name
	start := avgAnalysisSchedule.Start
	end := avgAnalysisSchedule.End
	duration := avgAnalysisSchedule.Duration
	mcod := multicycleonlydays

	// recover init values if existing
	MutexInitData.RLock()
	initC := InitData["sample__"][spacename][samplerName]
	initE := InitData["entry___"][spacename][samplerName]
	MutexInitData.RUnlock()

	samplerInterval := avgAnalysis[avgID].interval
	samplerIntervalMM := int64(samplerInterval) * 1000
	timeoutInterval := 5 * chantimeout * time.Millisecond
	if avgID > 0 {
		timeoutInterval += time.Duration(avgAnalysis[avgID-1].interval) * time.Second
	}
	counter := spaceEntries{ts: support.Timestamp(), val: 0}
	oldcounter := spaceEntries{ts: 0, val: 0}
	counter.entries = make(map[int]DataEntry)

	// update Start values if init values apply
	if initC != nil {
		if ts, err := strconv.ParseInt(initC[0], 10, 64); err == nil {
			if (counter.ts - ts) < Crashmaxdelay {
				if va, e := strconv.Atoi(initC[1]); e == nil {
					counter.ts = ts
					oldcounter.ts = ts
					counter.val = va
					oldcounter.val = va
					log.Printf("spaces.sampler: space %v loading sample recovery data for analysis %v\n", spacename, samplerName)
				}
			}
		}
	}

	if initE != nil {
		if ts, err := strconv.ParseInt(initE[0], 10, 64); err == nil {
			if (counter.ts - ts) < Crashmaxdelay {
				vas := strings.Split(initE[1][2:len(initE[1])-2], "][")
				var va [][]int
				for _, el := range vas {
					sd := strings.Split(el, " ")
					if len(sd) == 2 {
						sd0, e0 := strconv.Atoi(sd[0])
						sd1, e1 := strconv.Atoi(sd[1])
						if e0 == nil && e1 == nil {
							va = append(va, []int{sd0, sd1})
						}
					}
				}
				for _, j := range va {
					counter.entries[j[0]] = DataEntry{Val: j[1]}
				}
				log.Printf("spaces.sampler: space %v loading entry recovery data for analysis %v\n", spacename, samplerName)
			}
		}
	}

	support.DLog <- support.DevData{"counter starting " + spacename + samplerName,
		support.Timestamp(), "", []int{stats[0], stats[1]}, false}
	if prevStageChan == nil {
		log.Printf("spaces.sampler: error space %v not valid\n", spacename)
	} else {
		// this is the core
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"spaces.sampler: recovering server",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Printf("spaces.sampler: recovering for gate %+v from: %v\n ", prevStageChan, e)
				go sampler(spacename, prevStageChan, nextStageChan, syncPrevious, syncNext, avgID, once, tn, ntn)
			}
		}()

		log.Printf("spaces.sampler: setting sampler (%v,%v) for space %v\n", samplerName, samplerInterval, spacename)

		if avgID == 0 {
			// the first in the threads chain makes the counting
			// implements the current value as in sum over the period of time samplerInterval
			// it will always calculate the new sample, but only send it to the next stages in valid period (if defined)
			// or always.
			// It is also responsible to synchronising staged on Start and End period, if defined

			var cyclein bool
			firstDataOfCycle := true
			var maxOccupancy int
			if v, ok := SpaceMaxOccupancy[spacename]; ok {
				maxOccupancy = v
			} else {
				maxOccupancy = 0
			}
			// when there is no valid ANALYSISWINDOW, se let the samplers always run
			if duration == 0 {
				syncNext <- true
				cyclein = true
			} else {
				cyclein = false // this assignment is redundant but placed for readability
			}
			for {
				// wait till period starts and discard all values received
				// send sync
				// while in the period do as always
				select {
				case sp := <-prevStageChan:
					var skip bool
					var e error
					// in closure time the value is forced to zero
					if skip, e = support.InClosureTime(SpaceTimes[spacename].Start, SpaceTimes[spacename].End); e == nil {
						// reset counter in case we are in a CLOSURE period
						if skip {
							counter.val = 0
							// Calculate the confidence measurement (number wrong data / number data
							if sp.val != 0 {
								stats[0] += 1
								support.DLog <- support.DevData{"spaces.samplers counter " + spacename + " current",
									support.Timestamp(), "negative counter wrong/tots", []int{stats[0], stats[1]}, true}
							}
							sp.val = 0
						}
					}
					stats[1] += 1
					counter.val += sp.val
					// Calculate the confidence measurement (number wrong data / number data for DVL only
					if counter.val < 0 {
						stats[0] += 1
						support.DLog <- support.DevData{Tag: "spaces.samplers counter " + spacename + " current",
							Ts: support.Timestamp(), Note: "negative counter wrong/tots", Data: append([]int(nil), []int{stats[0], stats[1]}...), Aggr: true}
					}
					if counter.val < 0 && instNegSkip {
						counter.val = 0
					}
					if maxOccupancy != 0 {
						if counter.val > maxOccupancy {
							sp.val = 0
							counter.val = maxOccupancy
						}
					}
					// in closure time the value is forced to zero
					if e == nil {
						if skip {
							// reset all entries values in case we are in a CLOSURE period
							counter.entries[sp.id] = DataEntry{Val: 0}
						} else {
							if v, ok := counter.entries[sp.id]; ok {
								v.Val += sp.val
								counter.entries[sp.id] = v
							} else {
								counter.entries[sp.id] = DataEntry{Val: sp.val}
							}
						}
					}
				case <-time.After(timeoutInterval):
					if skip, e := support.InClosureTime(SpaceTimes[spacename].Start, SpaceTimes[spacename].End); e == nil {
						// reset all values in case we are in a CLOSURE period
						if skip {
							counter.val = 0
							for i := 0; i < len(counter.entries); i++ {
								counter.entries[i] = DataEntry{Val: 0}
							}
						}
					}
				}

				if duration != 0 {
					if incyc, e := support.InClosureTime(start, end); e != nil {
						log.Printf("spaces.sampler: error on InClosureTime for sampler (%v,%v) for space %v\n", samplerName, samplerInterval, spacename)
					} else {
						if !incyc && !cyclein {
							// was and stays out of cycle
							// does nothing, left for readability
						} else if !incyc && cyclein {
							// was in cycles and exits cycle
							// stops recording data and sendNext false
							syncNext <- false
							cyclein = false
						} else if incyc && cyclein {
							// was and stays in cycle
							// does nothing, left for readability
						} else {
							// was out and exnters cycke
							// starts recording and sends syncNext true
							syncNext <- true
							cyclein = true
							firstDataOfCycle = true

						}
					}
				}
				cTS := support.Timestamp()
				if (cTS - counter.ts) >= (int64(samplerInterval) * 1000) {
					counter.ts = cTS
					if support.Debug == -1 {
						fmt.Println(spacename, "current processing in ", cmode, " data ", counter)
					}
					if counter.val != oldcounter.val || oldcounter.ts == 0 || cmode == "0" || firstDataOfCycle {
						// new counter
						// data is stored according chantimeout selected compression mode CMODE
						// implements also CMODE 0 that only removes replicated values
						firstDataOfCycle = false
						oldcounter.val = counter.val
						oldcounter.ts = counter.ts
						oldcounter.entries = make(map[int]DataEntry)
						for i, v := range counter.entries {
							oldcounter.entries[i] = v
						}
						if cyclein {
							passData(spacename, samplerName, counter, nextStageChan, chantimeout, chantimeout)
						}
					} else if cmode == "1" {
						// counter did not change
						// if at least two entries have changed the sample is stored
						cd := 0
						for i, v := range counter.entries {
							if oldcounter.entries[i].Val != v.Val {
								cd += 1
							}
						}
						// at least two entries must have changed value
						if cd >= 2 {
							oldcounter.val = counter.val
							oldcounter.ts = counter.ts
							oldcounter.entries = make(map[int]DataEntry)
							for i, v := range counter.entries {
								oldcounter.entries[i] = v
							}
							if cyclein {
								passData(spacename, samplerName, counter, nextStageChan, chantimeout, chantimeout)
							}
						} else {
							if cd == 1 {
								support.DLog <- support.DevData{"spaces.samplers counter " + spacename + " current",
									support.Timestamp(), "inconsistent counter vs entries",
									[]int{}, true}
							}
						}
					} else if cmode == "2" {
						// counter did not change
						// verifies consistence of entry values in case of error
						// rejects or interpolates
						cd := 0
						count := counter.val
						for i, v := range counter.entries {
							count -= v.Val
							if oldcounter.entries[i].Val != v.Val {
								cd += 1
							}
						}
						// at least two entries must have changed value
						// and the counter is properly given by the entry values
						if (cd >= 2) && (count == 0) {
							oldcounter.val = counter.val
							oldcounter.ts = counter.ts
							oldcounter.entries = make(map[int]DataEntry)
							for i, v := range counter.entries {
								oldcounter.entries[i] = v
							}
							if cyclein {
								//fmt.Println("sending data in period")
								passData(spacename, samplerName, counter, nextStageChan, chantimeout, chantimeout)
							} else {
								//fmt.Println("skipping data out of period")
							}
							//passData(spacename, samplerName, counter, nextStageChan, chantimeout, chantimeout)
						} else {
							e := count != 0 || cd == 1
							if e {
								support.DLog <- support.DevData{"spaces.samplers counter " + spacename + " current",
									support.Timestamp(), "inconsistent counter vs entries",
									[]int{counter.val, count}, true}
							}
						}
					}
				}
			}
		} else {
			// threads 2+ in the chain needs to make the average and pass it forward
			var buffer []spaceEntries        // holds a cycle value
			var multiCycle []DataEntry       // holds averages of various cycles, Ts if not 0 gives a partial cycle
			leftincycle := samplerIntervalMM // keep track of a multicycle

			average := func(ct spaceEntries, buffer []spaceEntries, cTS int64, modifier int64) spaceEntries {
				if len(buffer) == 0 {
					return ct
				}
				period := float64(cTS - buffer[0].ts + modifier)
				acc := float64(0)
				if avgID == 1 {
					// first sampler need to consider sample permanence
					for i := 0; i < len(buffer)-1; i++ {
						acc += float64(buffer[i].val) * float64(buffer[i+1].ts-buffer[i].ts) / float64(period)
					}
				} else {
					// second and later sampler need to consider presence from past average
					for i := 1; i < len(buffer); i++ {
						acc += float64(buffer[i].val) * float64(buffer[i].ts-buffer[i-1].ts) / float64(period)
					}
				}
				// introduces an error in stages after the first. Its relevance depends on the ration between analysis periods
				acc += float64(buffer[len(buffer)-1].val) * float64(cTS-buffer[len(buffer)-1].ts) / float64(period)
				ct.val = int(math.Round(acc))
				if ct.val < 0 && avgNegSkip {
					ct.val = 0
				}

				// Extract all applicable series for each entry
				entries := make(map[int][]DataEntry)
				for i, v := range buffer {
					for j, ent := range v.entries {
						ent.Ts = buffer[i].ts
						entries[j] = append(entries[j], ent)
					}
				}

				// find latest value per entry
				ne := make(map[int]DataEntry)

				for i, entv := range entries {
					for j, ent := range entv {
						if j == 0 {
							ne[i] = ent
						} else {
							if ne[i].Ts < ent.Ts {
								ne[i] = ent
							}
						}
					}
				}

				ct.entries = ne
				ct.ts = cTS
				return ct
			}

			for {
				// wait till sync arrives, no samples will be received until then
				// store time of sync arrival
				// receive samples
				// when it is time for average, consider only de active hours
				// when sync for period arrives consider if sample will happen between periods (calculate and send now) or later
				// (adjust sample periodicity length). if periodicity sample smaller than period, simple reset and restart
				// it is needed to use a 2D structure with a line of data for every period to be considered for the average

				// get new data or timeout
				if startCycle := <-syncPrevious; startCycle {

					if syncNext != nil {
						syncNext <- true
					}

					if samplerInterval < 86400 {
						// the counter needs to be reset as we are not doing a multi cycle
						// thus we need sync reset at Start
						counter = spaceEntries{ts: support.Timestamp(), val: 0}
						buffer = []spaceEntries{counter}
					} else {
						buffer = []spaceEntries{{ts: support.Timestamp(), val: 0}}
					}

					// refts is used yo track multicycles situations that can End during a cycle.
					// it is bad practice, but it needs to be supported
					refts := int64(0)
					if support.Debug == -1 {
						fmt.Println(spacename, samplerName, "Start cycle", counter, buffer, "at", support.Timestamp())
					}

					for startCycle {

						avgsp := spaceEntries{ts: 0}
						select {
						case avgsp = <-prevStageChan:
							if support.Debug == -1 {
								fmt.Println(spacename, samplerName, "received", avgsp)
							}
						case <-time.After(timeoutInterval):
						case startCycle = <-syncPrevious:
							// This will never happen with Duration == 0
							if !startCycle {
								// check the various conditions of closure of the cycle

								if duration > samplerIntervalMM {
									// we need to just wait for next cycle
									// do nothing
									buffer = []spaceEntries{} // redundant
									if support.Debug == -1 {
										fmt.Println(spacename, samplerName, "End cycle do nothing")
									}
								} else {
									// we need to differentiate between spanning another cycle or ending in between
									// also considering the time used for a new analysis in the current cycle
									if samplerInterval < 86400 {
										// it is less than a day, so we calculate and send average
										// no data outside the cycle is used
										counter = average(counter, buffer, support.Timestamp(), samplerIntervalMM-duration)
										passData(spacename, samplerName, counter, nextStageChan, chantimeout,
											int(avgAnalysis[avgID-1].interval/2*1000))

										buffer = []spaceEntries{} // redundant
										if support.Debug == -1 {
											fmt.Println(spacename, samplerName, "End cycle do avg and send", counter)
										}

									} else {
										// it is a multi-cycle calculation
										// it can synchronise between cycles or be completely out of sync
										cTS := support.Timestamp()
										ct := average(counter, buffer, cTS, 0)
										ct.ts = refts
										multiCycle = append(multiCycle, DataEntry{Val: ct.val})
										if leftincycle > 86400000 {
											// we have more cycles
											// do nothing
											leftincycle -= 86400000 - (duration - refts)
											if support.Debug == -1 {
												fmt.Println(spacename, samplerName, "End cycle, multi non finished", leftincycle, counter)
											}
										} else {
											// the multi-cycle ends between cycles
											// we make average and send it out
											acc := float64(0)
											var nc float64
											if mcod {
												nc = math.Round(float64(samplerInterval) / 86400)
											} else {
												nc = float64(samplerInterval) / 86400
											}
											for _, sm := range multiCycle {
												if sm.Ts == 0 {
													acc += float64(sm.Val) / nc
												} else {
													acc += (float64(sm.Val) / nc) * (float64(sm.Ts) / float64(duration))
												}
											}
											counter.ts = cTS
											counter.val = int(math.Round(acc))
											passData(spacename, samplerName, counter, nextStageChan, chantimeout,
												int(avgAnalysis[avgID-1].interval/2*1000))
											leftincycle = samplerIntervalMM
											if support.Debug == -1 {

												fmt.Println(spacename, samplerName, "End cycly multi finished", mcod, counter)
											}
										}
										buffer = []spaceEntries{} // redundant
									}
								}
							} else {
								log.Printf("spaces.sampler: received sync:true instead of false for sampler (%v,%v) "+
									"for space %v\n", samplerName, samplerInterval, spacename)
							}
						}
						// if the time interval has passed a new sample is calculated and passed over
						if startCycle {
							cTS := support.Timestamp()
							if (cTS - counter.ts) >= samplerIntervalMM {
								if samplerInterval < 86400 {
									if buffer != nil {
										//average(buffer, cTS, samplerIntervalMM)
										counter = average(counter, buffer, cTS, 0)
										passData(spacename, samplerName, counter, nextStageChan, chantimeout,
											int(avgAnalysis[avgID-1].interval/2*1000))
										buffer[len(buffer)-1].ts = cTS
										buffer = append([]spaceEntries{}, buffer[len(buffer)-1])
									} else {
										// the following code will force the state to persist, it should not be reachable except
										// at the beginning of time
										counter.ts = cTS
										counter.val = 0
										buffer = append(buffer, counter)
										passData(spacename, samplerName, counter, nextStageChan, chantimeout, int(avgAnalysis[avgID-1].interval/2*1000))
									}
								} else {
									// multi-cycle analysis
									// we make the average and send it out
									ct := average(counter, buffer, cTS, 0)
									ct.ts = cTS
									multiCycle = append(multiCycle, DataEntry{Val: ct.val})
									acc := float64(0)
									nc := int(samplerInterval / 86400)
									for _, sm := range multiCycle {
										if sm.Ts == 0 {
											acc += float64(sm.Val / nc)
										} else {
											acc += float64(sm.Val/nc) * float64(sm.Ts/duration)
										}
									}
									counter.ts = cTS
									counter.val = int(math.Round(acc))
									passData(spacename, samplerName, counter, nextStageChan, chantimeout,
										int(avgAnalysis[avgID-1].interval/2*1000))
									// we prepare for the next sample Ts
									refts = duration - cTS
									buffer = append([]spaceEntries{}, buffer[len(buffer)-1])
									leftincycle = samplerIntervalMM
									if support.Debug == -1 {

										fmt.Println(spacename, samplerName, "avg calculation multi, restarts from", buffer)
									}
								}

							}
							//}
							// when not timed out, the new data is added to he queue
							// this data will also make the eventual average calculated irrelevant even if in the buffer (same cTS)
							if avgsp.ts != 0 {
								avgsp.ts = cTS
								buffer = append(buffer, avgsp)
							}
						}
					}
					if syncNext != nil {
						syncNext <- false
					}
				}
			}
		}
	}
}

// used internally in the sampler to pass data among threads.
func passData(spacename, samplerName string, counter spaceEntries, nextStageChan chan spaceEntries, stimeout, ltimeout int) {
	// need to make a new map to avoid pointer races
	cc := spaceEntries{id: counter.id, ts: counter.ts, val: counter.val}
	cc.entries = make(map[int]DataEntry)
	for i, v := range counter.entries {
		cc.entries[i] = v
	}
	// sending new data to the proper registers/DBS
	var wg sync.WaitGroup

	if support.Debug == -1 {
		fmt.Println(spacename, samplerName, "pass data:", cc)
	}

	latestChannelLock.RLock()
	for dl, dt := range dtypes {
		// only selects the sampler data, let generic for future extensions
		switch dl {
		case "presence":
			// skip
		default:
			wg.Add(1)
			data := dt.pf(dl+spacename+samplerName, cc)
			// new sample sent to the output register
			go func(data interface{}, ch chan interface{}) {
				defer wg.Done()
				select {
				case ch <- data:
				case <-time.After(time.Duration(stimeout) * time.Millisecond):
					log.Printf("spaces.samplers: Timeout writing to register for %v:%v\n", spacename, samplerName)
				}
			}(data, latestBankIn[dl][spacename][samplerName])
			// new sample sent to the database
			go func(data interface{}, ch chan interface{}) {
				// We do not need to wait for this goroutine
				select {
				case ch <- data:
				case <-time.After(time.Duration(ltimeout) * time.Millisecond):
					if support.Debug != 3 && support.Debug != 4 {
						log.Printf("spaces.samplers:: Timeout writing to sample database for %v:%v\n", spacename, samplerName)
					}
				}
			}(data, latestDBSIn[dl][spacename][samplerName])
		}
	}
	latestChannelLock.RUnlock()
	if nextStageChan != nil {
		select {
		case nextStageChan <- cc:
		case <-time.After(time.Duration(stimeout) * time.Millisecond):
			log.Printf("spaces.samplers: Timeout sending to next stage for %v:%v\n", spacename, samplerName)
		}
	}
	wg.Wait()
}
