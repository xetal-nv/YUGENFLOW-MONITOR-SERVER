package spaces

import (
	"countingserver/registers"
	"countingserver/support"
	"log"
	"time"
)

//func samplerold(spn string) {
//	c := spaceChannels[spn]
//	counter := 0
//	lastTS := support.Timestamp()
//	if c == nil {
//		log.Printf("spaces.sampler: error space %v not valid\n", spn)
//	} else {
//		log.Printf("spaces.sampler: enabled space [%v]\n", spn)
//		defer func() {
//			if e := recover(); e != nil {
//				if e != nil {
//					log.Printf("spaces.sampler: recovering for gate %+v from: %v\n ", c, e)
//					go sampler(spn)
//				}
//			}
//		}()
//		for {
//			// will need to add the check for groups and consensus
//			// when on group it stores in a groupo variable checking timestamos
//			// or only checks in the time out
//			select {
//			case val := <-c:
//				iv := int8(val.val)
//				if iv != 127 {
//					counter += int(iv)
//					if counter < 0 && negSkip {
//						counter = 0
//					}
//				}
//				cTS := support.Timestamp()
//				if (cTS - lastTS) >= (int64(samplingWindow) * 1000) {
//					LatestDataBankIn[spn]["current"] <- registers.DataCt{cTS, counter}
//					lastTS = cTS
//				}
//			default:
//				time.Sleep(100 * time.Millisecond)
//				cTS := support.Timestamp()
//				if (cTS - lastTS) >= (int64(samplingWindow) * 1000) {
//					LatestDataBankIn[spn]["current"] <- registers.DataCt{cTS, counter}
//					lastTS = cTS
//				}
//			}
//		}
//	}
//}

//func sampler(spn string) {
//	sampler(spn,0)
//}

// TODO the counter - in progress
// TODO need to see how to make the various counters
// TODO use a once and array with delays with the same code and difference channels?

func sampler(spacename string, avgID int) {
	c := spaceChannels[spacename]
	samplerName := avgAnalysis[avgID].name
	samplerInterval := avgAnalysis[avgID].interval
	timeoutInterval := 100 * time.Millisecond
	if avgID > 0 {
		timeoutInterval = time.Duration(avgAnalysis[avgID].interval) * time.Second
	}
	counter := 0
	lastTS := support.Timestamp()
	if c == nil {
		log.Printf("spaces.sampler: error space %v not valid\n", spacename)
	} else {
		//log.Printf("spaces.sampler: enabled space [%v]\n", spacename)
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Printf("spaces.sampler: recovering for gate %+v from: %v\n ", c, e)
					go sampler(spacename, avgID)
				}
			}
		}()
		log.Printf("spaces.sampler: setting sampler (%v,%v) for space %v\n", samplerName, samplerInterval, spacename)
		for {
			// will need to add the check for groups and consensus
			// when on group it stores in a groupo variable checking timestamos
			// or only checks in the time out
			select {
			case val := <-c:
				iv := int8(val.val)
				if iv != 127 {
					counter += int(iv)
					if counter < 0 && negSkip {
						counter = 0
					}
				}
				cTS := support.Timestamp()
				if (cTS - lastTS) >= (int64(samplerInterval) * 1000) {
					LatestDataBankIn[spacename][samplerName] <- registers.DataCt{cTS, counter}
					lastTS = cTS
				}
			default:
				time.Sleep(timeoutInterval)
				cTS := support.Timestamp()
				if (cTS - lastTS) >= (int64(samplerInterval) * 1000) {
					LatestDataBankIn[spacename][samplerName] <- registers.DataCt{cTS, counter}
					lastTS = cTS
				}
			}
		}
	}
}
