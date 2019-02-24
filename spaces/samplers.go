package spaces

import (
	"countingserver/support"
	"log"
	"sync"
	"time"
)

// TODO add counting per entry
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
	samplerInterval := avgAnalysis[avgID].interval
	timeoutInterval := 100 * time.Millisecond
	if avgID > 0 {
		timeoutInterval += time.Duration(avgAnalysis[avgID-1].interval) * time.Second
	}
	//counter := 0
	counter := spaceEntries{ts: support.Timestamp(), val: 0}
	support.DLog <- support.DevData{"counter starting " + spacename + samplerName, support.Timestamp(), "", stats}
	if prevStageChan == nil {
		log.Printf("spaces.sampler: error space %v not valid\n", spacename)
	} else {
		// this is the core
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Printf("spaces.sampler: recovering for gate %+v from: %v\n ", prevStageChan, e)
					go sampler(spacename, prevStageChan, nextStageChan, avgID, once, tn, ntn)
				}
			}
		}()

		log.Printf("spaces.sampler: setting sampler (%v,%v) for space %v\n", samplerName, samplerInterval, spacename)

		if avgID == 0 {
			// the first in the threads chain makes the counting
			for {
				cTS := support.Timestamp()
				select {
				case sp := <-prevStageChan:
					iv := int8(sp.val)
					stats[0] += 1
					counter.val += int(iv)
					if counter.val < 0 {
						// development logging
						stats[1] += 1
						support.DLog <- support.DevData{"counter " + spacename + " current", support.Timestamp(), "negative counter tot/negs", stats}
					}
					if counter.val < 0 && instNegSkip {
						counter.val = 0
					}

				default:
					time.Sleep(timeoutInterval)
				}
				if (cTS - counter.ts) >= (int64(samplerInterval) * 1000) {
					data := struct {
						Id  string
						ts  int64
						val int
					}{spacename + samplerName, cTS, counter.val}
					// new sample sent to the output registers
					go func() {
						//latestDataBankIn[spacename][samplerName] <- storage.DataCt{cTS, counter}
						latestDataBankIn[spacename][samplerName] <- data
					}()
					// new sample sent to the database
					go func() {
						//latestDataDBSIn[spacename][samplerName] <- storage.DataCt{cTS, counter}
						latestDataDBSIn[spacename][samplerName] <- data
					}()
					if nextStageChan != nil {
						counter.ts = cTS
						go func() { nextStageChan <- counter }()
					}
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
				default:
					time.Sleep(timeoutInterval)
				}
				cTS := support.Timestamp()
				// if the time interval has passed a new sample is calculated and passed over
				// TODO check
				if (cTS - counter.ts) >= (int64(samplerInterval) * 1000) {
					//fmt.Println("START", spacename, samplerName, cTS, counter.ts)
					//fmt.Println("START", spacename, samplerName, buffer)
					if buffer != nil {
						// when new samples have arrived we need to calculate the new state
						acc := int64(0)
						for i := 0; i < len(buffer)-1; i++ {
							acc += int64(buffer[i].val) * (buffer[i+1].ts - buffer[i].ts) / (cTS - buffer[0].ts)
							//fmt.Println(spacename, samplerName, i, acc)
						}
						acc += int64(buffer[len(buffer)-1].val) * (cTS - buffer[len(buffer)-1].ts) / (cTS - buffer[0].ts)
						//fmt.Println(spacename, samplerName, "final", acc)
						counter.val = int(acc)
						if counter.val < 0 && avgNegSkip {
							counter.val = 0
						}
						counter.ts = cTS
						data := struct {
							Id  string
							ts  int64
							val int
						}{spacename + samplerName, counter.ts, counter.val}
						// new sample sent to the output registers
						go func() {
							//latestDataBankIn[spacename][samplerName] <- storage.DataCt{cTS, avg}
							latestDataBankIn[spacename][samplerName] <- data
						}()
						// new sample sent to the database
						go func() {
							//latestDataDBSIn[spacename][samplerName] <- storage.DataCt{cTS, avg}
							latestDataDBSIn[spacename][samplerName] <- data
						}()
						if nextStageChan != nil {
							nextStageChan <- counter
						}

						buffer = nil
					} else {
						statsb[0] += 1
						support.DLog <- support.DevData{"counter " + spacename + samplerName, support.Timestamp(),
							"no samples branch count", statsb}
						// the following code will force the state to persist, it should not be reachable except
						// at the beginning of time
						counter.ts = cTS
						data := struct {
							Id  string
							ts  int64
							val int
						}{spacename + samplerName, counter.ts, counter.val}
						// new sample sent to the output registers
						go func() {
							//latestDataBankIn[spacename][samplerName] <- storage.DataCt{cTS, avg}
							latestDataBankIn[spacename][samplerName] <- data
						}()
						// new sample sent to the database
						go func() {
							//latestDataDBSIn[spacename][samplerName] <- storage.DataCt{cTS, avg}
							latestDataDBSIn[spacename][samplerName] <- data
						}()
						if nextStageChan != nil {
							nextStageChan <- counter
						}
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
