package spaces

import (
	"countingserver/support"
	"log"
	"math"
	"strconv"
	"sync"
	"time"
)

// TODO there is an issue when the servewr is started and NO sample has been sent!!!

// The algorithm is built on the ordered arrival of samples that is preserfved in a slice.
// It means that x[i] is newer than x[i-1] and older than x[i+1]
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
	timeoutInterval := chantimeout * time.Millisecond
	if avgID > 0 {
		timeoutInterval += time.Duration(avgAnalysis[avgID-1].interval) * time.Second
	}
	counter := spaceEntries{ts: support.Timestamp(), val: 0}
	counter.entries = make(map[int]dataEntry)
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
				select {
				case sp := <-prevStageChan:
					stats[0] += 1
					counter.val += sp.val
					if counter.val < 0 {
						stats[1] += 1
						support.DLog <- support.DevData{"counter " + spacename + " current", support.Timestamp(), "negative counter tot/negs", stats}
					}
					if counter.val < 0 && instNegSkip {
						counter.val = 0
					}
					if v, ok := counter.entries[sp.id]; ok {
						v.val += sp.val
						counter.entries[sp.id] = v
					} else {
						counter.entries[sp.id] = dataEntry{val: sp.val}
					}
					//fmt.Println("current step new sample", counter)
				case <-time.After(timeoutInterval):
				}
				cTS := support.Timestamp()
				if (cTS - counter.ts) >= (int64(samplerInterval) * 1000) {
					counter.ts = cTS
					passData(spacename, samplerName, counter, nextStageChan, chantimeout, chantimeout)
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
					//fmt.Println("received", samplerName, avgsp)
				case <-time.After(timeoutInterval):
				}
				cTS := support.Timestamp()
				// if the time interval has passed a new sample is calculated and passed over
				if (cTS - counter.ts) >= (int64(samplerInterval) * 1000) {
					if buffer != nil {
						// when new samples have arrived we need to calculate the new state
						acc := float64(0)
						for i := 0; i < len(buffer)-1; i++ {
							acc += float64(buffer[i].val) * float64(buffer[i+1].ts-buffer[i].ts) / float64(cTS-buffer[0].ts)
						}
						acc += float64(buffer[len(buffer)-1].val) * float64(cTS-buffer[len(buffer)-1].ts) / float64(cTS-buffer[0].ts)
						counter.val = int(math.RoundToEven(acc))
						if counter.val < 0 && avgNegSkip {
							counter.val = 0
						}

						// Extract all applicable seriesfor each entry
						entries := make(map[int][]dataEntry)
						ne := make(map[int]dataEntry)
						var wg sync.WaitGroup

						for i, v := range buffer {
							for j, ent := range v.entries {
								ent.ts = buffer[i].ts
								entries[j] = append(entries[j], ent)
							}
						}
						for i, v := range entries {
							wg.Add(1)
							go func(i int, v []dataEntry) {
								defer wg.Done()
								ne[i] = dataEntry{id: strconv.Itoa(i), ts: cTS, val: avgDataVector(v, cTS)}
							}(i, v)
						}
						wg.Wait()
						counter.entries = ne
						counter.ts = cTS
						passData(spacename, samplerName, counter, nextStageChan, chantimeout, int(avgAnalysis[avgID-1].interval/2*1000))
						buffer = nil
					} else {
						statsb[0] += 1
						support.DLog <- support.DevData{"counter " + spacename + samplerName, support.Timestamp(),
							"no samples branch count", statsb}
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

func avgDataVector(entries []dataEntry, cTS int64) (avg int) {

	acc := float64(0)
	for i := 0; i < len(entries)-1; i++ {
		acc += float64(entries[i].val) * float64(entries[i+1].ts-entries[i].ts) / float64(cTS-entries[0].ts)
	}
	acc += float64(entries[len(entries)-1].val) * float64(cTS-entries[len(entries)-1].ts) / float64(cTS-entries[0].ts)
	avg = int(math.RoundToEven(acc))
	return
}

func passData(spacename, samplerName string, counter spaceEntries, nextStageChan chan spaceEntries, stimeout, ltimeout int) {
	// need to make a new map to avoid pointer races
	cc := spaceEntries{id: counter.id, ts: counter.ts, val: counter.val}
	cc.entries = make(map[int]dataEntry)
	for i, v := range counter.entries {
		cc.entries[i] = v
	}
	// sending new data to the proper registers/DBS
	var wg sync.WaitGroup

	for n, dt := range dtypes {
		wg.Add(1)
		data := dt.pf(n+spacename+samplerName, cc)
		// new sample sent to the output register
		//fmt.Println("sending:", cc)
		go func(dtn string, data interface{}) {
			defer wg.Done()
			select {
			case latestBankIn[dtn][spacename][samplerName] <- data:
			case <-time.After(time.Duration(stimeout) * time.Millisecond):
				log.Printf("storage.passData: Timeout writing to register for %v:%v\n", spacename, samplerName)
			}
		}(n, data)
		// new sample sent to the database
		go func(dtn string, data interface{}) {
			// We do not need to wait for this goroutine
			select {
			case latestDBSIn[dtn][spacename][samplerName] <- data:
			case <-time.After(time.Duration(ltimeout) * time.Millisecond):
				if support.Debug != 3 && support.Debug != 4 {
					log.Printf("storage.passData: Timeout writing to sample database for %v:%v\n", spacename, samplerName)
				}
			}
		}(n, data)
	}
	if nextStageChan != nil {
		select {
		case nextStageChan <- cc:
		case <-time.After(time.Duration(stimeout) * time.Millisecond):
			log.Printf("storage.passData: Timeout sending to next stage for %v:%v\n", spacename, samplerName)
		}
	}
	wg.Wait()
}
