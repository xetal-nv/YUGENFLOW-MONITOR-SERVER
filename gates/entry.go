package gates

import (
	"countingserver/spaces"
	"countingserver/support"
	"fmt"
	"log"
)

func entryProcessing(id int, in chan sensorData) {
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				log.Printf("gates.entryProcessing: recovering for entry thread %v due to %v\n ", id, e)
				go entryProcessing(id, in)
			}
		}
	}()
	log.Printf("gates.entry: Processing: setting entry %v\n", id)
	for {
		data := <-in
		if support.Debug != 2 && support.Debug != 4 {
			trackPeople()
		} else {
			dp := data.val
			if e := spaces.SendData(id, dp); e != nil {
				log.Println(e)
			}
		}
		if support.Debug > 0 {
			fmt.Printf("DEBUG: entry %v has calculated datapoint at %v as %v\n", id, support.Timestamp(), data.val)
		}
	}

}

// TODO
func trackPeople() {
	// I do nothing right now!
}
