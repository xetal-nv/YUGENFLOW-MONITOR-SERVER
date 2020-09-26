package spaces

import (
	"fmt"
	"gateserver/supp"
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
// NOTE: for samples we take the time weighted averages, for entries the total (gets resetted only in closure time given by CLOSURE_ or on overflow)
// func sampler(spaceName string, prevStageChan, nextStageChan chan spaceEntries, syncPrevious, syncNext chan bool, avgID int, once sync.Once, tn, ntn int) {
func sampler(spaceName string, prevStageChan, nextStageChan chan spaceEntries, syncPrevious, syncNext chan bool, avgID, tn, ntn int, ct spaceEntries) {

	// // set-up the next analysis stage and the communication channel
	// once.Do(func() {
	// 	if avgID < (len(AvgAnalysis) - 1) {
	// 		nextStageChan = make(chan spaceEntries, bufferSize)
	// 		syncNext = make(chan bool)
	// 		go sampler(spaceName, nextStageChan, nil, syncNext, nil, avgID+1, 0, 0, sync.Once{})
	// 	}
	// })

	stats := []int{tn, ntn}
	//statsb := []int{0}
	samplerName := AvgAnalysis[avgID].Name
	start := avgAnalysisSchedule.Start
	end := avgAnalysisSchedule.End
	duration := avgAnalysisSchedule.Duration
	mcod := multiCycleOnlyDays

	// recover init values if existing
	MutexInitData.RLock()
	initC := InitData["sample__"][spaceName][samplerName]
	initE := InitData["entry___"][spaceName][samplerName]
	MutexInitData.RUnlock()

	samplerInterval := AvgAnalysis[avgID].Interval
	samplerIntervalMM := int64(samplerInterval) * 1000
	timeoutInterval := 5 * chanTimeout * time.Millisecond
	if avgID > 0 {
		timeoutInterval += time.Duration(AvgAnalysis[avgID-1].Interval) * time.Second
	}
	var counter spaceEntries
	oldCounter := spaceEntries{ts: 0, netFlow: 0}
	if ct.invalid {
		counter = spaceEntries{ts: supp.Timestamp(), netFlow: 0}
		counter.entries = make(map[int]DataEntry)
		// comment for debug: the code below in this block was before in a once statement
		// what follows is the set-up that should be run only the very first time
		if avgID < (len(AvgAnalysis) - 1) {
			nextStageChan = make(chan spaceEntries, bufferSize)
			syncNext = make(chan bool)
			go sampler(spaceName, nextStageChan, nil, syncNext, nil, avgID+1, 0, 0, spaceEntries{invalid: true})
		}
		// update Start values if init values apply
		if initC != nil {
			if ts, err := strconv.ParseInt(initC[0], 10, 64); err == nil {
				if (counter.ts - ts) < CrashMaxDelay {
					if va, e := strconv.Atoi(initC[1]); e == nil {
						counter.ts = ts
						oldCounter.ts = ts
						counter.netFlow = va
						oldCounter.netFlow = va
						log.Printf("spaces.sampler: space %v loading sample recovery data for analysis %v\n", spaceName, samplerName)
					}
				}
			}
		}
		if initE != nil {
			if ts, err := strconv.ParseInt(initE[0], 10, 64); err == nil {
				if (counter.ts - ts) < CrashMaxDelay {
					//if true {
					vas := strings.Split(initE[1][2:len(initE[1])-2], "][")
					var va [][]int
					for _, el := range vas {
						sd := strings.Split(el, " ")
						//fmt.Println(sd)
						if len(sd) == 4 {
							sd0, e0 := strconv.Atoi(sd[0])
							sd1, e1 := strconv.Atoi(sd[1])
							sd2, e2 := strconv.Atoi(sd[2])
							sd3, e3 := strconv.Atoi(sd[3])
							if e0 == nil && e1 == nil && e2 == nil && e3 == nil {
								va = append(va, []int{sd0, sd1, sd2, sd3})
							}
						}
						//fmt.Println(sd,va)
					}
					for _, j := range va {
						counter.entries[j[0]] = DataEntry{Ts: ts, NetFlow: j[1], PositiveFlow: j[2], NegativeFlow: j[3]}
					}
					log.Printf("spaces.sampler: space %v loading entry recovery data for analysis %v\n", spaceName, samplerName)
				}
			}
		}
	} else {
		counter = ct
	}

	supp.DLog <- supp.DevData{"counter starting " + spaceName + samplerName,
		supp.Timestamp(), "", []int{stats[0], stats[1]}, false}
	if prevStageChan == nil {
		log.Printf("spaces.sampler: error space %v not valid\n", spaceName)
	} else {
		// this is the core part of the sampler
		// recovery on can only be enabled at this point when all set-up has passsed safely
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"spaces.sampler: recovering server",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Printf("spaces.sampler: recovering for gate %+v from: %v\n ", prevStageChan, e)
				go sampler(spaceName, prevStageChan, nextStageChan, syncPrevious, syncNext, avgID, stats[0], stats[1], counter)
			}
		}()

		// log is updated only at first start, not after a recovery
		if ct.invalid {
			log.Printf("spaces.sampler: starting sampler (%v,%v) for space %v\n", samplerName, samplerInterval, spaceName)
		}

		if avgID == 0 {
			// the first in the threads chain makes the counting
			// implements the current value as in sum over the period of time samplerInterval
			// it will always calculate the new sample, but will only send it to the next stages in valid period (if defined)
			// or always.
			// It is also responsible to synchronising all stages on Start and End period, if defined

			var cycleIn bool
			firstDataOfCycle := true
			var maxOccupancy int
			if v, ok := SpaceMaxOccupancy[spaceName]; ok {
				maxOccupancy = v
			} else {
				maxOccupancy = 0
			}
			// when there is no valid ANALYSISWINDOW, we let the samplers always run
			if duration == 0 {
				syncNext <- true
				cycleIn = true
			} else {
				cycleIn = false // this assignment is redundant but placed for readability
			}
			for {
				// wait till period starts and discard all values received
				// send sync
				// while in the period do as always
				select {
				case sp := <-prevStageChan:
					// var skip bool
					// var e error
					// in closure time the value is forced to zero
					if skip, e := supp.InClosureTime(SpaceTimes[spaceName].Start, SpaceTimes[spaceName].End); e == nil {
						if skip {
							// fmt.Println(spaceName, samplerName, "counter at skip", counter)
							// reset counter, we are in a CLOSURE period
							counter.netFlow = 0
							// counter.entries[sp.id] = DataEntry{NetFlow: 0, PositiveFlow: 0, NegativeFlow: 0}
							// for i := 0; i < len(counter.entries); i++ {
							// 	counter.entries[i] = DataEntry{counter.entries[i].id, counter.entries[i].Ts, 0, 0, 0}
							// }
							for i, el := range counter.entries {
								el.NegativeFlow = 0
								el.NetFlow = 0
								el.PositiveFlow = 0
								counter.entries[i] = el
							}
							// Calculate the confidence measurement (number wrong data / number data
							if sp.netFlow != 0 {
								supp.DLog <- supp.DevData{"spaces.samplers counter " + spaceName + " current",
									supp.Timestamp(), "not zero flow received during closure time", []int{0}, true}
							}
							// sp.netFlow = 0
						} else {
							// fmt.Println(spaceName, samplerName, "counter outside skip", counter)
							// calculate new flows in case of not closure time
							stats[1] += 1
							// overflow should never happen at this point, however we add a check since it can be the signal of a system instability
							// counter.netFlow += sp.netFlow
							sum, of := supp.CheckIntOverflow(counter.netFlow, sp.netFlow)
							counter.netFlow = sum
							// fmt.Println(spaceName, counter.netFlow, sp.netFlow)
							if of {
								supp.DLog <- supp.DevData{"spaces.samplers counter " + spaceName + " current",
									supp.Timestamp(), "overflow on counter", []int{0}, true}
							}
							// Calculate the confidence measurement (number wrong data / number data for DVL only
							if counter.netFlow < 0 {
								stats[0] += 1
								supp.DLog <- supp.DevData{Tag: "spaces.samplers counter " + spaceName + " current",
									Ts: supp.Timestamp(), Note: "negative counter wrong/tots", Data: append([]int(nil), []int{stats[0], stats[1]}...), Aggr: true}
							}
							// adjust flow based on conditions set for negative and maximum flows from the configuration file
							if counter.netFlow < 0 && instNegSkip {
								counter.netFlow = 0
								sp.netFlow = 0
							}
							if maxOccupancy != 0 {
								if counter.netFlow > maxOccupancy {
									// we calculate the flow that brought us to maxOccupancy
									sp.netFlow = sp.netFlow - (counter.netFlow - maxOccupancy)
									counter.netFlow = maxOccupancy
								}
							}
							if v, ok := counter.entries[sp.id]; ok {
								// overflow should never happen at this point, however we add a check since it can be the signal of a system instability
								// v.NetFlow += sp.netFlow
								sum, of := supp.CheckIntOverflow(v.NetFlow, sp.netFlow)
								v.NetFlow = sum
								if of {
									supp.DLog <- supp.DevData{"spaces.samplers counter.entries  " + strconv.Itoa(sp.id),
										supp.Timestamp(), "overflow on netflow counter", []int{0}, true}
								}

								// over and underflow needs to force reset of values with the correct difference
								// this can happen when no closure time has been defined
								if sp.netFlow > 0 {
									sum, of := supp.CheckIntOverflow(v.PositiveFlow, sp.netFlow)
									v.PositiveFlow = sum
									if of {
										v.NegativeFlow = 0
									}
								} else {
									sum, of := supp.CheckIntOverflow(v.NegativeFlow, sp.netFlow)
									v.NegativeFlow = sum
									if of {
										v.PositiveFlow = 0
									}
								}
								counter.entries[sp.id] = v
							} else {
								if sp.netFlow > 0 {
									counter.entries[sp.id] = DataEntry{NetFlow: sp.netFlow, PositiveFlow: sp.netFlow}
								} else {
									counter.entries[sp.id] = DataEntry{NetFlow: sp.netFlow, NegativeFlow: sp.netFlow}
								}
							}
						}
					}
				case <-time.After(timeoutInterval):
					if skip, e := supp.InClosureTime(SpaceTimes[spaceName].Start, SpaceTimes[spaceName].End); e == nil {
						// reset all values in case we are in a CLOSURE period
						if skip {
							// fmt.Println(spaceName, samplerName, "counter at skip and to", counter)
							counter.netFlow = 0
							for i, el := range counter.entries {
								el.NegativeFlow = 0
								el.NetFlow = 0
								el.PositiveFlow = 0
								counter.entries[i] = el
							}
							// for i := 0; i < len(counter.entries); i++ {
							// 	counter.entries[i] = DataEntry{counter.entries[i].id, counter.entries[i].Ts, 0, 0, 0}
							// }
							// fmt.Println(spaceName, samplerName, "counter after skip and to", counter)
						}
					}
				}

				// handles period sync with all related analysis (if needed)
				if duration != 0 {
					if inCycle, e := supp.InClosureTime(start, end); e != nil {
						log.Printf("spaces.sampler: error on sync InClosureTime for sampler (%v,%v) for space %v\n", samplerName, samplerInterval, spaceName)
					} else {
						if !inCycle && !cycleIn {
							// was and stays out of cycle
							// does nothing, left for readability
						} else if !inCycle && cycleIn {
							// was in cycles and exits cycle
							// stops recording data and sendNext false
							syncNext <- false
							cycleIn = false
						} else if inCycle && cycleIn {
							// was and stays in cycle
							// does nothing, left for readability
						} else {
							// was out and enter cycle
							// starts recording and sends syncNext true
							syncNext <- true
							cycleIn = true
							firstDataOfCycle = true

						}
					}
				}

				// handles the finalisation of a sample for a complete sampler interval
				cTS := supp.Timestamp()
				if (cTS - counter.ts) >= (int64(samplerInterval) * 1000) {
					counter.ts = cTS
					if supp.Debug == -1 {
						fmt.Println(spaceName, "current processing in ", compressionMode, " for data ", counter)
					}
					if counter.netFlow != oldCounter.netFlow || oldCounter.ts == 0 || compressionMode == "0" || firstDataOfCycle {
						// new counter
						// data is stored according chanTimeout selected compression mode CMODE
						// implements also CMODE 0 that only removes replicated values
						firstDataOfCycle = false
						oldCounter.netFlow = counter.netFlow
						oldCounter.ts = counter.ts
						oldCounter.entries = make(map[int]DataEntry)
						for i, v := range counter.entries {
							oldCounter.entries[i] = v
						}
						if cycleIn {
							passData(spaceName, samplerName, counter, nextStageChan, chanTimeout, chanTimeout)
						}
					} else if compressionMode == "1" {
						// counter did not change
						// if at least two entries have changed the sample is stored
						cd := 0
						for i, v := range counter.entries {
							if oldCounter.entries[i].NetFlow != v.NetFlow {
								cd += 1
							}
						}
						// at least two entries must have changed value
						if cd >= 2 {
							oldCounter.netFlow = counter.netFlow
							oldCounter.ts = counter.ts
							oldCounter.entries = make(map[int]DataEntry)
							for i, v := range counter.entries {
								oldCounter.entries[i] = v
							}
							if cycleIn {
								passData(spaceName, samplerName, counter, nextStageChan, chanTimeout, chanTimeout)
							}
						} else {
							if cd == 1 {
								supp.DLog <- supp.DevData{"spaces.samplers counter " + spaceName + " current",
									supp.Timestamp(), "inconsistent counter vs entries",
									[]int{0}, true}
							}
						}
					} else if compressionMode == "2" {
						// counter did not change
						// verifies consistence of entry values in case of error
						// rejects or interpolates
						cd := 0
						count := counter.netFlow
						for i, v := range counter.entries {
							count -= v.NetFlow
							if oldCounter.entries[i].NetFlow != v.NetFlow {
								cd += 1
							}
						}
						// at least two entries must have changed value
						// and the counter is properly given by the entry values
						if (cd >= 2) && (count == 0) {
							oldCounter.netFlow = counter.netFlow
							oldCounter.ts = counter.ts
							oldCounter.entries = make(map[int]DataEntry)
							for i, v := range counter.entries {
								oldCounter.entries[i] = v
							}
							if cycleIn {
								//fmt.Println("sending data in period")
								passData(spaceName, samplerName, counter, nextStageChan, chanTimeout, chanTimeout)
							} else {
								//fmt.Println("skipping data out of period")
							}
							//passData(spaceName, samplerName, counter, nextStageChan, chanTimeout, chanTimeout)
						} else {
							e := count != 0 || cd == 1
							if e {
								supp.DLog <- supp.DevData{"spaces.samplers counter " + spaceName + " current",
									supp.Timestamp(), "inconsistent counter vs entries",
									[]int{counter.netFlow, count}, true}
							}
						}
					}
				}
			}
		} else {
			// threads 2+ in the chain needs to make the average and pass it forward
			var buffer []spaceEntries        // holds a cycle value
			var multiCycle []DataEntry       // holds averages of various cycles, Ts if not 0 gives a partial cycle
			leftInCycle := samplerIntervalMM // keep track of a multi cycle

			// calculates the average presence and the cumulative (or latest) entry values (or flow)
			average := func(ct spaceEntries, buffer []spaceEntries, cTS int64, modifier int64) spaceEntries {
				//fmt.Println("==>>",spaceName, samplerName, ct, buffer);
				if len(buffer) == 0 {
					return ct
				}
				period := float64(cTS - buffer[0].ts + modifier)
				acc := float64(0)
				if avgID == 1 {
					// first sampler need to consider sample permanence
					for i := 0; i < len(buffer)-1; i++ {
						//noinspection GoRedundantConversion
						acc += float64(buffer[i].netFlow) * float64(buffer[i+1].ts-buffer[i].ts) / float64(period)
					}
				} else {
					// second and later sampler need to consider presence from past average
					for i := 1; i < len(buffer); i++ {
						//noinspection GoRedundantConversion
						acc += float64(buffer[i].netFlow) * float64(buffer[i].ts-buffer[i-1].ts) / float64(period)
					}
				}
				// introduces an error in stages after the first. Its relevance depends on the ration between analysis periods
				//noinspection GoRedundantConversion
				acc += float64(buffer[len(buffer)-1].netFlow) * float64(cTS-buffer[len(buffer)-1].ts) / float64(period)
				ct.netFlow = int(math.Round(acc))
				if ct.netFlow < 0 && avgNegSkip {
					ct.netFlow = 0
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

				//fmt.Println("==>>",spaceName, samplerName, ct);
				return ct
			}

			for {
				// This loop:
				//  - wait till sync arrives, no samples will be received until then
				//  - store time of sync arrival
				//  - receive samples
				//  - when it is time for average, consider only the active hours
				//  - when sync for period arrives consider if sample will happen between periods (calculate and send now) or later
				//  - adjust sample periodicity length. if periodicity sample smaller than period, simple reset and restart
				//  - it is needed to use a 2D structure with a line of data for every period to be considered for the average

				// get new data or timeout
				if startCycle := <-syncPrevious; startCycle {

					if syncNext != nil {
						syncNext <- true
					}

					if samplerInterval < 86400 {
						// the counter needs to be reset as we are not doing a multi cycle
						// thus we need sync reset at Start
						counter = spaceEntries{ts: supp.Timestamp(), netFlow: 0}
						buffer = []spaceEntries{counter}
					} else {
						buffer = []spaceEntries{{ts: supp.Timestamp(), netFlow: 0}}
					}

					// refTS is used yo track multicycles situations that can End during a cycle.
					// it is bad practice, but it needs to be supported
					refTS := int64(0)
					if supp.Debug == -1 {
						fmt.Println(spaceName, samplerName, "Start cycle", counter, buffer, "at", supp.Timestamp())
					}

					for startCycle {

						avgSP := spaceEntries{ts: 0}
						select {
						case avgSP = <-prevStageChan:
							if supp.Debug == -1 {
								fmt.Println(spaceName, samplerName, "received", avgSP)
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
									if supp.Debug == -1 {
										fmt.Println(spaceName, samplerName, "End cycle do nothing")
									}
								} else {
									// we need to differentiate between spanning another cycle or ending in between
									// also considering the time used for a new analysis in the current cycle
									if samplerInterval < 86400 {
										// it is less than a day, so we calculate and send average
										// no data outside the cycle is used
										counter = average(counter, buffer, supp.Timestamp(), samplerIntervalMM-duration)
										passData(spaceName, samplerName, counter, nextStageChan, chanTimeout,
											AvgAnalysis[avgID-1].Interval/2*1000)

										buffer = []spaceEntries{} // redundant
										if supp.Debug == -1 {
											fmt.Println(spaceName, samplerName, "End cycle do avg and send", counter)
										}

									} else {
										// it is a multi-cycle calculation
										// it can synchronise between cycles or be completely out of sync
										cTS := supp.Timestamp()
										ct := average(counter, buffer, cTS, 0)
										ct.ts = refTS
										multiCycle = append(multiCycle, DataEntry{NetFlow: ct.netFlow})
										if leftInCycle > 86400000 {
											// we have more cycles
											// do nothing
											leftInCycle -= 86400000 - (duration - refTS)
											if supp.Debug == -1 {
												fmt.Println(spaceName, samplerName, "End cycle, multi non finished", leftInCycle, counter)
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
													acc += float64(sm.NetFlow) / nc
												} else {
													acc += (float64(sm.NetFlow) / nc) * (float64(sm.Ts) / float64(duration))
												}
											}
											counter.ts = cTS
											counter.netFlow = int(math.Round(acc))
											passData(spaceName, samplerName, counter, nextStageChan, chanTimeout,
												AvgAnalysis[avgID-1].Interval/2*1000)
											leftInCycle = samplerIntervalMM
											if supp.Debug == -1 {

												fmt.Println(spaceName, samplerName, "End multi cycle finished", mcod, counter)
											}
										}
										buffer = []spaceEntries{} // redundant
									}
								}
							} else {
								log.Printf("spaces.sampler: received sync:true instead of false for sampler (%v,%v) "+
									"for space %v\n", samplerName, samplerInterval, spaceName)
							}
						}
						// if the time interval has passed a new sample is calculated and passed over
						if startCycle {
							cTS := supp.Timestamp()
							if (cTS - counter.ts) >= samplerIntervalMM {
								if samplerInterval < 86400 {
									if buffer != nil {
										//average(buffer, cTS, samplerIntervalMM)
										counter = average(counter, buffer, cTS, 0)
										passData(spaceName, samplerName, counter, nextStageChan, chanTimeout,
											AvgAnalysis[avgID-1].Interval/2*1000)
										buffer[len(buffer)-1].ts = cTS
										buffer = append([]spaceEntries{}, buffer[len(buffer)-1])
									} else {
										// the following code will force the state to persist, it should not be reachable except
										// at the beginning of time
										counter.ts = cTS
										counter.netFlow = 0
										buffer = append(buffer, counter)
										passData(spaceName, samplerName, counter, nextStageChan, chanTimeout, AvgAnalysis[avgID-1].Interval/2*1000)
									}
								} else {
									// multi-cycle analysis
									// we make the average and send it out
									ct := average(counter, buffer, cTS, 0)
									ct.ts = cTS
									multiCycle = append(multiCycle, DataEntry{NetFlow: ct.netFlow})
									acc := float64(0)
									nc := samplerInterval / 86400
									for _, sm := range multiCycle {
										if sm.Ts == 0 {
											acc += float64(sm.NetFlow / nc)
										} else {
											acc += float64(sm.NetFlow/nc) * float64(sm.Ts/duration)
										}
									}
									counter.ts = cTS
									counter.netFlow = int(math.Round(acc))
									passData(spaceName, samplerName, counter, nextStageChan, chanTimeout,
										AvgAnalysis[avgID-1].Interval/2*1000)
									// we prepare for the next sample Ts
									refTS = duration - cTS
									buffer = append([]spaceEntries{}, buffer[len(buffer)-1])
									leftInCycle = samplerIntervalMM
									if supp.Debug == -1 {

										fmt.Println(spaceName, samplerName, "avg calculation multi, restarts from", buffer)
									}
								}

							}
							//}
							// when not timed out, the new data is added to he queue
							// this data will also make the eventual average calculated irrelevant even if in the buffer (same cTS)
							if avgSP.ts != 0 {
								avgSP.ts = cTS
								buffer = append(buffer, avgSP)
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
func passData(spaceName, samplerName string, counter spaceEntries, nextStageChan chan spaceEntries, sTimeout, lTimeout int) {

	shadowSingleMux.RLock()
	if shadowAnalysis != "" { // this check is redundant
		if samplerName == shadowAnalysis {
			today := time.Now().Format("01-02-2006")
			if shadowAnalysisDate[spaceName] != today {
				if shadowAnalysisDate[spaceName] != "" {
					if _, e := shadowAnalysisFile[spaceName].WriteString("\n"); e != nil {
						log.Printf("ShadowReport error writing new line for %v, at date %v\n", spaceName, today)
					}
				}
				if _, e := shadowAnalysisFile[spaceName].WriteString(time.Now().Format("2006.01.02") +
					", (" + time.Now().Format("15:04") + ", " + strconv.Itoa(counter.netFlow) + ")"); e != nil {
					log.Printf("ShadowReport error writing first line for %v, at date %v\n", spaceName, today)
				}
				shadowAnalysisDate[spaceName] = time.Now().Format("01-02-2006")
			} else {
				if _, e := shadowAnalysisFile[spaceName].WriteString(", (" + time.Now().Format("15:04") + ", " + strconv.Itoa(counter.netFlow) + ")"); e != nil {
					log.Printf("ShadowReport error writing new value for %v, at time %v\n", spaceName, time.Now().Format("2006.01.02 15:04:05"))
				}
			}
			//fmt.Println(spacename, samplerName, shadowAnalysisDate[spacename])
		}
	}
	shadowSingleMux.RUnlock()
	// need to make a new map to avoid pointer races
	cc := spaceEntries{id: counter.id, ts: counter.ts, netFlow: counter.netFlow}
	cc.entries = make(map[int]DataEntry)
	for i, v := range counter.entries {
		cc.entries[i] = v
	}
	// sending new data to the proper registers/DBS
	var wg sync.WaitGroup

	if supp.Debug == -1 {
		fmt.Println(spaceName, samplerName, "pass data:", cc)
	}

	//fmt.Println(spaceName, samplerName, "pass data:", cc)

	latestChannelLock.RLock()
	for dl, dt := range dataTypes {
		// only selects the sampler data, let generic for future extensions
		switch dl {
		case "presence":
			// skip
		default:
			wg.Add(2)
			data := dt.pf(dl+spaceName+samplerName, cc)
			// new sample sent to the output register
			go func(data interface{}, ch chan interface{}) {
				defer wg.Done()
				select {
				case ch <- data:
				case <-time.After(time.Duration(sTimeout) * time.Millisecond):
					log.Printf("spaces.samplers: Timeout writing to register for %v:%v\n", spaceName, samplerName)
				}
			}(data, latestBankIn[dl][spaceName][samplerName])
			// new sample sent to the database
			go func(data interface{}, ch chan interface{}) {
				defer wg.Done()
				select {
				case ch <- data:
				case <-time.After(time.Duration(lTimeout) * time.Millisecond):
					//if supp.Debug != 3 && supp.Debug != 4 {
					log.Printf("spaces.samplers:: Timeout writing to sample database for %v:%v\n", spaceName, samplerName)
					//}
					// fmt.Printf("spaces.samplers:: Timeout writing to sample database for %v:%v\n", spaceName, samplerName)
				}
			}(data, latestDBSIn[dl][spaceName][samplerName])
		}
	}
	latestChannelLock.RUnlock()
	if nextStageChan != nil {
		select {
		case nextStageChan <- cc:
		case <-time.After(time.Duration(sTimeout) * time.Millisecond):
			log.Printf("spaces.samplers: Timeout sending to next stage for %v:%v\n", spaceName, samplerName)
		}
	}
	wg.Wait()
}
