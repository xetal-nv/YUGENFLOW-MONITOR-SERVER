package spaces

import (
	"countingserver/registers"
	"countingserver/support"
	"log"
	"time"
)

// TODO the counter - in progress
// TODO need to see how to make the various counters
// TODO use a once and array with delays with the same code and difference channels?
func sampler(spn string) {
	//time.Sleep(1 * time.Second)
	//saveToFile(spn)
	c := spaceChannels[spn]
	counter := 0
	lastTS := support.Timestamp()
	if c == nil {
		log.Printf("spaces.sampler: error space %v not valid\n", spn)
	} else {
		log.Printf("spaces.sampler: enabled space [%v]\n", spn)
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Printf("spaces.sampler: recovering for gate %+v from: %v\n ", c, e)
					go sampler(spn)
				}
			}
		}()
		for {
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
				if (cTS - lastTS) >= (int64(samplingWindow) * 1000) {
					LatestDataBankIn[spn]["current"] <- registers.DataCt{cTS, counter}
					//fmt.Printf("%v :: counter for %v is %v\n", support.Timestamp(), spn, counter)
					lastTS = cTS
				}
			default:
				time.Sleep(100 * time.Millisecond)
				cTS := support.Timestamp()
				if (cTS - lastTS) >= (int64(samplingWindow) * 1000) {
					LatestDataBankIn[spn]["current"] <- registers.DataCt{cTS, counter}
					//fmt.Printf("%v :: counter for %v is %v\n", support.Timestamp(), spn, counter)
					lastTS = cTS
				}
			}
		}
	}
}
