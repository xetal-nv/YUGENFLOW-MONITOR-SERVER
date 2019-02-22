package spaces

import (
	"countingserver/support"
	"log"
	"sync"
	"time"
)

func sampler(spacename string, prevStageChan, nextStageChan chan interface{}, avgID int, once sync.Once, tn, ntn int) {
	// set-up the next analysis stage and the communication channel
	once.Do(func() {
		if avgID < (len(avgAnalysis) - 1) {
			nextStageChan = make(chan interface{}, bufsize)
			go sampler(spacename, nextStageChan, nil, avgID+1, sync.Once{}, 0, 0)
		}
	})
	stats := []int{tn, ntn}
	samplerName := avgAnalysis[avgID].name
	samplerInterval := avgAnalysis[avgID].interval
	timeoutInterval := 100 * time.Millisecond
	if avgID > 0 {
		timeoutInterval += time.Duration(avgAnalysis[avgID-1].interval) * time.Second
	}
	counter := 0
	lastTS := support.Timestamp()
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
				case psc := <-prevStageChan:
					val := new(dataEntry)
					if e := val.Extract(psc); e != nil {
						log.Println(e, psc)
					} else {
						iv := int8(val.val)
						stats[0] += 1
						counter += int(iv)
						if counter < 0 {
							// development logging
							stats[1] += 1
							support.DLog <- support.DevData{"counter " + spacename + " current", support.Timestamp(), "negative counter", stats}
						}
						if counter < 0 && instNegSkip {
							counter = 0
						}
					}
				default:
					time.Sleep(timeoutInterval)
				}
				if (cTS - lastTS) >= (int64(samplerInterval) * 1000) {
					data := struct {
						Id  string
						ts  int64
						val int
					}{spacename + samplerName, cTS, counter}
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
						go func() { nextStageChan <- dataEntry{val: counter, ts: cTS} }()
					}
					lastTS = cTS
				}
			}
		} else {
			// threads 2+ in the chain needs to make the average and pass it forward
			var buffer []dataEntry
			for {
				// get new data or timeout
				cTS := support.Timestamp()
				val := new(dataEntry)
				valid := true
				select {
				case psc := <-prevStageChan:
					if e := val.Extract(psc); e != nil {
						valid = false
					}
				default:
					time.Sleep(timeoutInterval)
					valid = false
				}
				// if the time interval has passed a new sample is calculated and passed over
				if (cTS - lastTS) >= (int64(samplerInterval) * 1000) {
					go func(cTS, lastTS int64) {
						acc := int64(0)
						refTS := lastTS
						for _, v := range buffer {
							sp := int64(v.val)
							if sp < 0 && avgNegSkip {
								sp = 0
							}
							acc += sp * (v.ts - refTS)
						}
						avg := int(acc / (cTS - lastTS))
						data := struct {
							Id  string
							ts  int64
							val int
						}{spacename + samplerName, cTS, avg}
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
							nextStageChan <- dataEntry{val: avg, ts: cTS}
						}
					}(cTS, lastTS)

					buffer = nil
					lastTS = cTS
				}
				// when not timed out, the new data is added to he queue
				if valid {
					buffer = append(buffer, *val)
				}
			}
		}
	}
}
