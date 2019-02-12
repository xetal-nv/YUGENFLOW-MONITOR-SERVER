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
		//if resultf, e = os.OpenFile(spn+".csv", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); e != nil {
		if resultf, e = os.OpenFile(spn+".csv", os.O_WRONLY|os.O_CREATE, 0644); e != nil { // DEBUG
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
			dt := 0
			if val.val == 255 {
				dt = -1
			}
			if val.val == 1 {
				dt = 1
			}
			sp := ""
			for i := 0; i < val.num; i++ {
				sp += "0,"
			}
			//if _, e := fmt.Fprintln(resultf, val.group, ",", support.Timestamp(), spaceChannels[1:], dt); e != nil {
			if _, e := fmt.Fprintln(resultf, support.Timestamp(), sp[1:], dt); e != nil {
				log.Printf("apaces.saveToFile: error space %v not valid\n", spn)
			}
		}
	}
}
