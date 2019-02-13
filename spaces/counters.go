package spaces

import (
	"countingserver/support"
	"fmt"
	"log"
)

// TODO the counter - in progress
// TODO how to we do the counting? double windows/queue?
func Counters(spn string) {
	//time.Sleep(1 * time.Second)
	//saveToFile(spn)
	c := spaceChannels[spn]
	counter := 0
	lastTS := support.Timestamp()
	if c == nil {
		log.Printf("spaces.Counters: error space %v not valid\n", spn)
	} else {
		log.Printf("spaces.Counters: enabled space [%v]\n", spn)
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Printf("spaces.Counters: recovering for gate %+v from: %v\n ", c, e)
					go Counters(spn)
				}
			}
		}()
		for {
			val := <-c
			iv := int8(val.val)
			if iv != 127 {
				counter += int(iv)
				if counter < 0 {
					counter = 0
				}
			}
			cTS := support.Timestamp()
			if (cTS - lastTS) >= (samplingWindow * 1000) {
				fmt.Printf("%v :: counter for %v is %v\n", support.Timestamp(), spn, counter)
				lastTS = cTS
			}
		}
	}
}
