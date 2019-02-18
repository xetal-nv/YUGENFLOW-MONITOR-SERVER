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
	log.Printf("gates.entryProcessing: setting entry %v\n", id)
	for {
		data := <-in
		// TODO to be done fully, now it just sends the data it receives
		dp := data.val
		if e := spaces.SendData(id, dp); e != nil {
			log.Println(e)
		}
		fmt.Printf("DEBUG: entry %v has calculated datapoint at %v as %v\n", id, support.Timestamp(), dp)
	}

}
