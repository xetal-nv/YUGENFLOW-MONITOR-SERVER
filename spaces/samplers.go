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

// TODO add usage of time schedule for averages ONLY!

// it implements the counters, both the current one as well as the analysis averages.
// the sampler threads for a given space are started in a recursive manner
// The algorithm is built on the ordered arrival of samples that is preserved in a slice.
// It means that x[i] is newer than x[i-1] and older than x[i+1]
// NOTE: for samples we take the time weighted averages, for entries the total only on the analysis period
func sampler(spacename string, prevStageChan, nextStageChan chan spaceEntries, avgID int, once sync.Once, tn, ntn int) {
	// set-up the next analysis stage and the communication channel
	once.Do(func() {
		if avgID < (len(avgAnalysis) - 1) {
			nextStageChan = make(chan spaceEntries, bufsize)
			go sampler(spacename, nextStageChan, nil, avgID+1, sync.Once{}, 0, 0)
		}
	})

	stats := []int{tn, ntn}
	statsb := []int{0}
	samplerName := avgAnalysis[avgID].name

	// recover init values if existing
	MutexInitData.RLock()
	initC := InitData["sample__"][spacename][samplerName]
	initE := InitData["entry___"][spacename][samplerName]
	MutexInitData.RUnlock()

	samplerInterval := avgAnalysis[avgID].interval
	timeoutInterval := chantimeout * time.Millisecond
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
				go sampler(spacename, prevStageChan, nextStageChan, avgID, once, tn, ntn)
			}
		}()

		log.Printf("spaces.sampler: setting sampler (%v,%v) for space %v\n", samplerName, samplerInterval, spacename)

		offset := 0
		updateCount := func(to int) {
			if support.Debug == -1 {
				fmt.Println(spacename, "current processing in ", cmode, " data ", counter)
			}
			if counter.val != oldcounter.val || oldcounter.ts == 0 || cmode == "0" {
				// new counter
				// data is stored according to selected compression mode CMODE
				// implements also CMODE 0 that onlt removes replicated values
				sd := true
				if cmode == "3" {
					// Experimental interpolation mode, not to use in release
					count := counter.val
					for _, v := range counter.entries {
						count -= v.val
					}
					if count != 0 {
						// we distribute the errors only if it repeats and stays the same
						if count == offset {
							for i := 0; i < support.Abs(count); i++ {
								tmp := counter.entries[i%len(counter.entries)]
								if count > 0 {
									tmp.val += 1
								} else {
									tmp.val -= 1
								}
								counter.entries[i%len(counter.entries)] = tmp
							}
						} else {
							offset = count
							sd = false
						}
					}
				}
				if sd {
					oldcounter.val = counter.val
					oldcounter.ts = counter.ts
					oldcounter.entries = make(map[int]dataEntry)
					for i, v := range counter.entries {
						oldcounter.entries[i] = v
					}
					passData(spacename, samplerName, counter, nextStageChan, chantimeout, to)
				}
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
					passData(spacename, samplerName, counter, nextStageChan, chantimeout, to)
				} else {
					if cd == 1 {
						support.DLog <- support.DevData{"spaces.samplers counter " + spacename + " current",
							support.Timestamp(), "inconsistent counter vs entries",
							[]int{}, true}
					}
				}
			} else if cmode == "2" || cmode == "3" {
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
					passData(spacename, samplerName, counter, nextStageChan, chantimeout, to)
				} else {
					e := count != 0 || cd == 1
					if count != 0 && cd >= 2 && cmode == "3" {

						if count == offset {
							// we distribute the errors only if it repeats and stays the same
							e = false
							for i := 0; i < support.Abs(count); i++ {
								tmp := counter.entries[i%len(counter.entries)]
								if count > 0 {
									tmp.val += 1
								} else {
									tmp.val -= 1
								}
								counter.entries[i%len(counter.entries)] = tmp
							}
							oldcounter.val = counter.val
							oldcounter.ts = counter.ts
							oldcounter.entries = make(map[int]dataEntry)
							for i, v := range counter.entries {
								oldcounter.entries[i] = v
							}
							passData(spacename, samplerName, counter, nextStageChan, chantimeout, to)
						} else {
							offset = count
							e = true
						}
					}
					if e {
						support.DLog <- support.DevData{"spaces.samplers counter " + spacename + " current",
							support.Timestamp(), "inconsistent counter vs entries",
							[]int{counter.val, count}, true}
					}
				}
			}
		}

		if avgID == 0 {
			// the first in the threads chain makes the counting
			// implements the current value as in sum over the period of time samplerInterval
			for {
				select {
				case sp := <-prevStageChan:
					var skip bool
					var e error
					// in closure time the value is forced to zero
					if skip, e = support.InClosureTime(spaceTimes[spacename].start, spaceTimes[spacename].end); e == nil {
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
					// in closure time the value is forced to zero
					if e == nil {
						if skip {
							counter.entries[sp.id] = dataEntry{val: 0}
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
				}
				//fmt.Println(counter)
				cTS := support.Timestamp()
				if (cTS - counter.ts) >= (int64(samplerInterval) * 1000) {
					counter.ts = cTS
					updateCount(chantimeout)
				}
			}
		} else {
			// threads 2+ in the chain needs to make the average and pass it forward
			var buffer []spaceEntries
			for {
				// get new data or timeout
				avgsp := spaceEntries{ts: 0}
				select {
				case avgsp = <-prevStageChan:
					if support.Debug == -1 {
						fmt.Println(spacename, samplerName, "received", avgsp)
					}
				case <-time.After(timeoutInterval):
				}
				cTS := support.Timestamp()
				// if the time interval has passed a new sample is calculated and passed over
				if (cTS - counter.ts) >= (int64(samplerInterval) * 1000) {
					if buffer != nil {
						//fmt.Println(avgID, buffer)
						// a weighted average is calculate based on sample permanence
						acc := float64(0)
						for i := 0; i < len(buffer)-1; i++ {
							acc += float64(buffer[i].val) * float64(buffer[i+1].ts-buffer[i].ts) / float64(cTS-buffer[0].ts)
						}
						acc += float64(buffer[len(buffer)-1].val) * float64(cTS-buffer[len(buffer)-1].ts) / float64(cTS-buffer[0].ts)
						counter.val = int(math.RoundToEven(acc))
						if counter.val < 0 && avgNegSkip {
							counter.val = 0
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

						//fmt.Println(entries, ne)

						//nec := make(chan struct {
						//	id   int
						//	data dataEntry
						//}, len(entries))
						//for i, v := range entries {
						//	go func(i int, v []dataEntry) {
						//		nec <- struct {
						//			id   int
						//			data dataEntry
						//		}{i, dataEntry{id: strconv.Itoa(i), ts: cTS, val: avgDataVector(v, cTS)}}
						//	}(i, v)
						//}
						//for range entries {
						//	el := <-nec
						//	ne[el.id] = el.data
						//}
						counter.entries = ne
						counter.ts = cTS
						if cstats == "1" {
							updateCount(int(avgAnalysis[avgID-1].interval / 2 * 1000))
						} else {
							passData(spacename, samplerName, counter, nextStageChan, chantimeout, int(avgAnalysis[avgID-1].interval/2*1000))
						}
						buffer = nil
					} else {
						statsb[0] += 1
						support.DLog <- support.DevData{"spaces.samplers counter " + spacename + samplerName, support.Timestamp(),
							"no samples branch count", statsb, true}
						// the following code will force the state to persist, it should not be reachable except
						// at the beginning of time
						counter.ts = cTS
						passData(spacename, samplerName, counter, nextStageChan, chantimeout, int(avgAnalysis[avgID-1].interval/2*1000))
					}
				}
				// when not timed out, the new data is added to he queue
				if avgsp.ts != 0 {
					avgsp.ts = cTS
					buffer = append(buffer, avgsp)
				} else if buffer == nil {
					// the first sample of the series is the previous result
					buffer = append(buffer, counter)
				}
			}
		}
	}
}

// calculates the average data from a vector conteining value and timestamps
//func avgDataVector(entries []dataEntry, cTS int64) (avg int) {
//
//	acc := float64(0)
//	for i := 0; i < len(entries)-1; i++ {
//		acc += float64(entries[i].val) * float64(entries[i+1].ts-entries[i].ts) / float64(cTS-entries[0].ts)
//	}
//	acc += float64(entries[len(entries)-1].val) * float64(cTS-entries[len(entries)-1].ts) / float64(cTS-entries[0].ts)
//	avg = int(math.RoundToEven(acc))
//	return
//}

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
		//go func(dtn string, data interface{}) {
		//	defer wg.Done()
		//	select {
		//	case latestBankIn[dtn][spacename][samplerName] <- data:
		//	case <-time.After(time.Duration(stimeout) * time.Millisecond):
		//		log.Printf("spaces.samplers: Timeout writing to register for %v:%v\n", spacename, samplerName)
		//	}
		//}(n, data)
		//// new sample sent to the database
		//go func(dtn string, data interface{}) {
		//	// We do not need to wait for this goroutine
		//	select {
		//	case latestDBSIn[dtn][spacename][samplerName] <- data:
		//	case <-time.After(time.Duration(ltimeout) * time.Millisecond):
		//		if support.Debug != 3 && support.Debug != 4 {
		//			log.Printf("spaces.samplers:: Timeout writing to sample database for %v:%v\n", spacename, samplerName)
		//		}
		//	}
		//}(n, data)
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
