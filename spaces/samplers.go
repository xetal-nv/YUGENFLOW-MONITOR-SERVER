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

// TODO add usage of time schedule for averages ONLY

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
	start := avgAnalysisSchedule.start
	end := avgAnalysisSchedule.end
	duration := avgAnalysisSchedule.duration
	mcod := multicycleonlydays

	//schedule := !(start == time.Time{})
	//fmt.Println(duration, avgID, avgAnalysis[avgID].interval)

	//fmt.Println(samplerName, avgAnalysisSchedule.duration/(int64(avgAnalysis[avgID].interval)*1000))

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
	counter.entries = make(map[int]dataEntry)

	// update start values if init values apply
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
					counter.entries[j[0]] = dataEntry{val: j[1]}
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
			// It is also responsible to synchronising staged on start and end period, if defined

			var cyclein bool
			firstDataOfCycle := true
			var maxOccupancy int
			if v, ok := SpaceMaxOccupancy[spacename]; ok {
				maxOccupancy = v
			} else {
				maxOccupancy = 0
			}
			//fmt.Println(maxOccupancy)
			// when there is no valid ANALYSISWINDOW, se let the samplers always run
			if duration == 0 {
				//fmt.Println("no duration is defined")
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
					if skip, e = support.InClosureTime(spaceTimes[spacename].start, spaceTimes[spacename].end); e == nil {
						// reset counter in case we are in a CLOSURE period
						//fmt.Println(skip)
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
							//fmt.Println("got above limit", sp.val, counter)
						}
					}
					// in closure time the value is forced to zero
					if e == nil {
						if skip {
							// reset all entries values in case we are in a CLOSURE period
							counter.entries[sp.id] = dataEntry{val: 0}
							fmt.Println("reset", counter)
						} else {
							if v, ok := counter.entries[sp.id]; ok {
								v.val += sp.val
								counter.entries[sp.id] = v
							} else {
								counter.entries[sp.id] = dataEntry{val: sp.val}
							}
						}
					}
				case <-time.After(timeoutInterval):
					if skip, e := support.InClosureTime(spaceTimes[spacename].start, spaceTimes[spacename].end); e == nil {
						// reset all values in case we are in a CLOSURE period
						//fmt.Println(skip)
						if skip {
							counter.val = 0
							for i := 0; i < len(counter.entries); i++ {
								counter.entries[i] = dataEntry{val: 0}
							}
							fmt.Println("reset", counter)
						}
					}
				}
				//fmt.Println(counter)
				if duration != 0 {
					if incyc, e := support.InClosureTime(start, end); e != nil {
						//fmt.Println(" !!! error on InClosureTime")
						log.Printf("spaces.sampler: error on InClosureTime for sampler (%v,%v) for space %v\n", samplerName, samplerInterval, spacename)
					} else {
						if !incyc && !cyclein {
							//fmt.Println("was out, stays out")
							// does nothing, remove
						} else if !incyc && cyclein {
							//fmt.Println("was in, goes out")
							// stops recording data and sendNext false
							syncNext <- false
							cyclein = false
						} else if incyc && cyclein {
							//fmt.Println("was in, stays in")
							// does nothing, remove
						} else {
							//fmt.Println("was out, goes in")
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
					//offset := 0
					//updateCount := func(chantimeout int) {
					if support.Debug == -1 {
						fmt.Println(spacename, "current processing in ", cmode, " data ", counter)
					}
					if counter.val != oldcounter.val || oldcounter.ts == 0 || cmode == "0" || firstDataOfCycle {
						// new counter
						// data is stored according chantimeout selected compression mode CMODE
						// implements also CMODE 0 that only removes replicated values
						//sd := true
						//if cmode == "3" {
						//	// Experimental interpolation mode, not chantimeout use in release
						//	count := counter.val
						//	for _, v := range counter.entries {
						//		count -= v.val
						//	}
						//	if count != 0 {
						//		// we distribute the errors only if it repeats and stays the same
						//		if count == offset {
						//			for i := 0; i < support.Abs(count); i++ {
						//				tmp := counter.entries[i%len(counter.entries)]
						//				if count > 0 {
						//					tmp.val += 1
						//				} else {
						//					tmp.val -= 1
						//				}
						//				counter.entries[i%len(counter.entries)] = tmp
						//			}
						//		} else {
						//			offset = count
						//			sd = false
						//		}
						//	}
						//}
						//if sd {
						firstDataOfCycle = false
						oldcounter.val = counter.val
						oldcounter.ts = counter.ts
						oldcounter.entries = make(map[int]dataEntry)
						for i, v := range counter.entries {
							oldcounter.entries[i] = v
						}
						if cyclein {
							//fmt.Println("sending data in period")
							passData(spacename, samplerName, counter, nextStageChan, chantimeout, chantimeout)
						} else {
							//fmt.Println("skipping data out of period")
						}
						//}
					} else if cmode == "1" {
						// counter did not change
						// if at least two entries have changed the sample is stored
						cd := 0
						for i, v := range counter.entries {
							if oldcounter.entries[i].val != v.val {
								cd += 1
							}
						}
						// at least two entries must have changed value
						if cd >= 2 {
							oldcounter.val = counter.val
							oldcounter.ts = counter.ts
							oldcounter.entries = make(map[int]dataEntry)
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
							if cd == 1 {
								support.DLog <- support.DevData{"spaces.samplers counter " + spacename + " current",
									support.Timestamp(), "inconsistent counter vs entries",
									[]int{}, true}
							}
						}
						//} else if cmode == "2" || cmode == "3" {
					} else if cmode == "2" {
						// counter did not change
						// verifies consistence of entry values in case of error
						// rejects or interpolates
						cd := 0
						count := counter.val
						for i, v := range counter.entries {
							count -= v.val
							if oldcounter.entries[i].val != v.val {
								cd += 1
							}
						}
						// at least two entries must have changed value
						// and the counter is properly given by the entry values
						if (cd >= 2) && (count == 0) {
							oldcounter.val = counter.val
							oldcounter.ts = counter.ts
							oldcounter.entries = make(map[int]dataEntry)
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
							//if count != 0 && cd >= 2 && cmode == "3" {
							//
							//	if count == offset {
							//		// we distribute the errors only if it repeats and stays the same
							//		e = false
							//		for i := 0; i < support.Abs(count); i++ {
							//			tmp := counter.entries[i%len(counter.entries)]
							//			if count > 0 {
							//				tmp.val += 1
							//			} else {
							//				tmp.val -= 1
							//			}
							//			counter.entries[i%len(counter.entries)] = tmp
							//		}
							//		oldcounter.val = counter.val
							//		oldcounter.ts = counter.ts
							//		oldcounter.entries = make(map[int]dataEntry)
							//		for i, v := range counter.entries {
							//			oldcounter.entries[i] = v
							//		}
							//		if cyclein {
							//			//fmt.Println("sending data in period")
							//			passData(spacename, samplerName, counter, nextStageChan, chantimeout, chantimeout)
							//		} else {
							//			//fmt.Println("skipping data out of period")
							//		}
							//		//passData(spacename, samplerName, counter, nextStageChan, chantimeout, chantimeout)
							//	} else {
							//		offset = count
							//		e = true
							//	}
							//}
							if e {
								support.DLog <- support.DevData{"spaces.samplers counter " + spacename + " current",
									support.Timestamp(), "inconsistent counter vs entries",
									[]int{counter.val, count}, true}
							}
						}
					}
					//}

					//updateCount(chantimeout)
				}
				// when the period is over send the sync ang back to the start
			}
		} else {
			// threads 2+ in the chain needs to make the average and pass it forward
			var buffer []spaceEntries  // holds a cycle value
			var multiCycle []dataEntry // holds averages of various cycles, ts if not 0 gives a partial cycle
			//currentCycle := samplerIntervalMM // current cycle duration of a multicycle
			leftincycle := samplerIntervalMM // keep track of a multicycle

			//average := func(buffer []spaceEntries, cTS int64, period int64) {
			average := func(ct spaceEntries, buffer []spaceEntries, cTS int64, modifier int64) spaceEntries {
				//fmt.Println(period, cTS-buffer[0].ts)
				//if buffer != nil {
				//fmt.Println(avgID, buffer)
				if len(buffer) == 0 {
					return ct
				}
				period := float64(cTS - buffer[0].ts + modifier)
				acc := float64(0)
				if avgID == 1 {
					fmt.Println(samplerName, buffer)
					// first sampler need to consider sample permanence
					for i := 0; i < len(buffer)-1; i++ {
						//acc += float64(buffer[i].val) * float64(buffer[i+1].ts-buffer[i].ts) / float64(cTS-buffer[0].ts)
						acc += float64(buffer[i].val) * float64(buffer[i+1].ts-buffer[i].ts) / float64(period)
						fmt.Println(samplerName, acc)
					}
				} else {
					// second and later sampler need to consider presence from past average
					fmt.Println(samplerName, buffer)
					for i := 1; i < len(buffer); i++ {
						//acc += float64(buffer[i].val) * float64(buffer[i].ts-buffer[i-1].ts) / float64(cTS-buffer[0].ts)
						acc += float64(buffer[i].val) * float64(buffer[i].ts-buffer[i-1].ts) / float64(period)
						fmt.Println(samplerName, acc)
					}
				}
				// introduces an error in stages after the first. Its relevance depends on the ration between analysis periods
				//acc += float64(buffer[len(buffer)-1].val) * float64(cTS-buffer[len(buffer)-1].ts) / float64(cTS-buffer[0].ts)
				acc += float64(buffer[len(buffer)-1].val) * float64(cTS-buffer[len(buffer)-1].ts) / float64(period)
				ct.val = int(math.Round(acc))
				//if avgID == 1 {
				fmt.Println(samplerName, ct.val)
				//}
				if ct.val < 0 && avgNegSkip {
					ct.val = 0
				}

				// Extract all applicable series for each entry
				entries := make(map[int][]dataEntry)
				for i, v := range buffer {
					for j, ent := range v.entries {
						ent.ts = buffer[i].ts
						entries[j] = append(entries[j], ent)
					}
				}

				// find latest value per entry
				ne := make(map[int]dataEntry)

				for i, entv := range entries {
					for j, ent := range entv {
						if j == 0 {
							ne[i] = ent
						} else {
							if ne[i].ts < ent.ts {
								ne[i] = ent
							}
						}
					}
				}

				ct.entries = ne
				ct.ts = cTS
				return ct
				//passData(spacename, samplerName, ct, nextStageChan, chantimeout, int(avgAnalysis[avgID-1].interval/2*1000))
				//buffer = nil
				//} else {
				//	statsb[0] += 1
				//	support.DLog <- support.DevData{"spaces.samplers ct " + spacename + samplerName, support.Timestamp(),
				//		"no samples branch count", statsb, true}
				//	// the following code will force the state to persist, it should not be reachable except
				//	// at the beginning of time
				//	ct.ts = cTS
				//	passData(spacename, samplerName, ct, nextStageChan, chantimeout, int(avgAnalysis[avgID-1].interval/2*1000))
				//}
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

					//fmt.Println(spacename, samplerName, "new cycle started")
					if syncNext != nil {
						syncNext <- true
					}

					if samplerInterval < 86400 {
						// the counter needs to be reset as we are not doing a multi cycle
						// thus we need sync reset at start
						counter = spaceEntries{ts: support.Timestamp(), val: 0}
						buffer = []spaceEntries{counter}
					} else {
						buffer = []spaceEntries{spaceEntries{ts: support.Timestamp(), val: 0}}
					}

					// refts is used yo track multicycles situations that can end during a cycle.
					// it is bad practice, but it needs to be supported
					refts := int64(0)
					//buffer = []spaceEntries{counter} // redundant
					//startTSCycle := counter.ts
					fmt.Println(spacename, samplerName, "start cycle", counter, buffer, "at", support.Timestamp())

					for startCycle {

						avgsp := spaceEntries{ts: 0}
						select {
						case avgsp = <-prevStageChan:
							if support.Debug == -1 {
								fmt.Println(spacename, samplerName, "received", avgsp)
							}
						case <-time.After(timeoutInterval):
						case startCycle = <-syncPrevious:
							// This will never happen with duration == 0
							if !startCycle {
								//fmt.Println(spacename, samplerName, "cycle stopped")
								// check the various conditions of closure of the cycle

								if duration > samplerIntervalMM {
									// we need to just wait for next cycle
									// do nothing
									buffer = []spaceEntries{} // redundant
									fmt.Println(spacename, samplerName, "end cycle do nothing")
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
										fmt.Println(spacename, samplerName, "end cycle do avg and send", counter)

									} else {
										// it is a multi-cycle calculation
										// it can synchronise between cycles or be completely out of sync
										cTS := support.Timestamp()
										ct := average(counter, buffer, cTS, 0)
										ct.ts = refts
										multiCycle = append(multiCycle, dataEntry{val: ct.val})
										if leftincycle > 86400000 {
											// we have more cycles
											// do nothing
											leftincycle -= 86400000 - (duration - refts)
											fmt.Println(spacename, samplerName, "end cycle, multi non finished", leftincycle, counter)

										} else {
											// the multi-cycle ends between cycles
											// we make average and send it out
											acc := float64(0)
											//nc := math.RoundToEven(float64(samplerInterval) / 86400)
											var nc float64
											if mcod {
												nc = math.Round(float64(samplerInterval) / 86400)
											} else {
												nc = float64(samplerInterval) / 86400
											}
											for _, sm := range multiCycle {
												if sm.ts == 0 {
													acc += float64(sm.val) / nc
												} else {
													acc += (float64(sm.val) / nc) * (float64(sm.ts) / float64(duration))
												}
											}
											counter.ts = cTS
											counter.val = int(math.Round(acc))
											passData(spacename, samplerName, counter, nextStageChan, chantimeout,
												int(avgAnalysis[avgID-1].interval/2*1000))
											leftincycle = samplerIntervalMM
											fmt.Println(spacename, samplerName, "end cycly multi finished", mcod, counter)
										}
										buffer = []spaceEntries{} // redundant
									}
								}
							} else {
								fmt.Println(spacename, samplerName, "error in end of cycle sync")
								log.Printf("spaces.sampler: received sync:true instead of false for sampler (%v,%v) "+
									"for space %v\n", samplerName, samplerInterval, spacename)
							}
						}
						// if the time interval has passed a new sample is calculated and passed over
						if startCycle {
							cTS := support.Timestamp()
							if (cTS - counter.ts) >= samplerIntervalMM {
								if samplerInterval < 86400 {
									fmt.Println(spacename, samplerName, "avg calculation")
									if buffer != nil {
										//average(buffer, cTS, samplerIntervalMM)
										counter = average(counter, buffer, cTS, 0)
										passData(spacename, samplerName, counter, nextStageChan, chantimeout,
											int(avgAnalysis[avgID-1].interval/2*1000))
										buffer[len(buffer)-1].ts = cTS
										buffer = append([]spaceEntries{}, buffer[len(buffer)-1])
									} else {
										//statsb[0] += 1
										//support.DLog <- support.DevData{"spaces.samplers counter " + spacename + samplerName, support.Timestamp(),
										//	"no samples branch count", statsb, true}
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
									multiCycle = append(multiCycle, dataEntry{val: ct.val})
									acc := float64(0)
									nc := int(samplerInterval / 86400)
									for _, sm := range multiCycle {
										if sm.ts == 0 {
											acc += float64(sm.val / nc)
										} else {
											acc += float64(sm.val/nc) * float64(sm.ts/duration)
										}
									}
									counter.ts = cTS
									counter.val = int(math.Round(acc))
									passData(spacename, samplerName, counter, nextStageChan, chantimeout,
										int(avgAnalysis[avgID-1].interval/2*1000))
									// we prepare for the next sample ts
									refts = duration - cTS
									buffer = append([]spaceEntries{}, buffer[len(buffer)-1])
									leftincycle = samplerIntervalMM
									fmt.Println(spacename, samplerName, "avg calculation multi, restarts from", buffer)
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
						//} else if buffer == nil {
						//	// the first sample of the series is the previous result
						//	buffer = append(buffer, counter)
						//}
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
	cc.entries = make(map[int]dataEntry)
	for i, v := range counter.entries {
		cc.entries[i] = v
	}
	// sending new data to the proper registers/DBS
	var wg sync.WaitGroup

	if support.Debug == -1 {
		fmt.Println(spacename, samplerName, "pass data:", cc)
	}

	latestChannelLock.RLock()
	for n, dt := range dtypes {
		wg.Add(1)
		data := dt.pf(n+spacename+samplerName, cc)
		// new sample sent to the output register
		go func(data interface{}, ch chan interface{}) {
			defer wg.Done()
			select {
			case ch <- data:
			case <-time.After(time.Duration(stimeout) * time.Millisecond):
				log.Printf("spaces.samplers: Timeout writing to register for %v:%v\n", spacename, samplerName)
			}
		}(data, latestBankIn[n][spacename][samplerName])
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
		}(data, latestDBSIn[n][spacename][samplerName])
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
