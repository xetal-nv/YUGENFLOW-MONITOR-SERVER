package spaces

import (
	"countingserver/support"
	"fmt"
	"log"
	"os"
)

// saves raw data to a file
func saveToFile(spn string) {
	c := spaceChannels[spn]
	if c == nil {
		log.Printf("spaces.saveToFile: error space %v not valid\n", spn)
	} else {
		log.Printf("spaces.saveToFile: enabled space [%v]\n", spn)
		var resultf *os.File
		var e error

		if os.Getenv("DELDATAFILES") == "1" {
			//noinspection GoUnhandledErrorResult
			os.Remove(spn + ".csv")
		}

		if resultf, e = os.OpenFile(spn+".csv", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); e != nil { // DEBUG
			log.Fatal(e)
		}

		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Printf("spaces.saveToFile: recovering for gate %+v from: %v\n ", c, e)
					//noinspection GoUnhandledErrorResult
					resultf.Close()
					go saveToFile(spn)
				}
			}
		}()
		for {
			val := <-c
			if _, e := fmt.Fprintln(resultf, support.Timestamp(), ",", val.num, ",", int8(val.val),
				",", val.group); e != nil {
				log.Printf("apaces.saveToFile: error space %v not valid\n", spn)
			}
		}
	}
}
