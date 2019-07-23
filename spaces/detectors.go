package spaces

import (
	"gateserver/storage"
	"gateserver/support"
	"log"
	"os"
	"strings"
	"time"
)

type intervalDetector struct {
	id         string    // entry id as string to support entry data in the entire communication pipe
	start, end time.Time // start and end of the interval
	incycle    bool      // track if it si in cycle
	activity   dataEntry // activity count
}

//func detectors(spacename string, prevStageChan, nextStageChan chan spaceEntries, syncPrevious, syncNext chan bool, avgID int, once sync.Once, tn, ntn int) {
func detectors(name string, gateChan chan spaceEntries, allIntervals []intervalDetector,
	sendDBSchan map[string]chan interface{}, tn, ntn int) {

	//latestDBSIn[dl][name][v.name] = make(chan interface{})
	//label := dl + name + v.name
	//if _, e := storage.SetSeries(label, v.interval, !support.Stringending(label, "current", "_")); e != nil {
	//	log.Fatal("spaces.setUpDataDBSBank: fatal error setting database %v:%v\n", name+v.name, v.interval)
	//}
	//go dt.cf(dl+name+v.name, latestDBSIn[dl][name][v.name], nil)

	timeoutInterval := 5 * chantimeout * time.Millisecond
	active := true
	// verify if configuration is already given
	if len(allIntervals) == 0 {
		// configuration is read, if name is truncated, the name needs to be recovered from the config file as well
		var spacename string
		if name[len(name)-1] != '_' {
			// we need to recovery the name since truncated
			sps := strings.Split(strings.Trim(os.Getenv("SPACES_NAMES"), " "), " ")
			for _, nm := range sps {
				nmck := strings.Trim(strings.Split(nm, ":")[0], " ")
				if name == nmck[0:len(name)] {
					spacename = nmck
				}
			}
		} else {
			spacename = strings.Trim(name, "_")
		}
		if sts := os.Getenv("PRESENCE_" + spacename); sts != "" {
			// all intervals are read
			for _, st := range strings.Split(strings.Trim(sts, " "), ";") {
				stdata := strings.Split(strings.Trim(st, " "), " ")
				if start, e := time.Parse(support.TimeLayout, strings.Trim(stdata[1], " ")); e == nil {
					if end, e := time.Parse(support.TimeLayout, strings.Trim(stdata[2], " ")); e == nil {
						spacename = support.StringLimit(spacename, support.LabelLength)
						nm := support.StringLimit("presence", support.LabelLength) + spacename + support.StringLimit(strings.Trim(stdata[0], " "), support.LabelLength)
						allIntervals = append(allIntervals, intervalDetector{nm,
							start, end, false, dataEntry{id: nm}})
						sendDBSchan[nm] = make(chan interface{})
						//label := spacename + nm
						if _, e := storage.SetSeries(nm, SamplingWindow, true); e != nil {
							log.Fatalf("spaces.detectors: fatal error setting database %v\n", nm)
						}
						go dtypes["presence"].cf(nm, sendDBSchan[nm], nil)
					}
				}
			}

		} else {
			active = false
			//fmt.Println("detector for", name, "not active")
		}
	}
	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"spaces.detector: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			log.Printf("spaces.detectors: space %v detector recovering from : %v\n ", name, e)
			go detectors(name, gateChan, allIntervals, sendDBSchan, tn, ntn)
		}
	}()

	//fmt.Println(sendDBSchan)

	for {
		var sp spaceEntries
		select {
		case sp = <-gateChan:
		case <-time.After(timeoutInterval):
		}
		if active {
			// check for proper activity falling in an interval and save at the end of the interval
			// we have some activity
			for i := range allIntervals {
				//fmt.Println("checking", allIntervals[i].id)
				if found, e := support.InClosureTime(allIntervals[i].start, allIntervals[i].end); e == nil && found {
					allIntervals[i].incycle = true
					if sp.val != 0 {
						allIntervals[i].activity.val += 1
						allIntervals[i].activity.ts = support.Timestamp()
						//fmt.Println("space activity for interval", allIntervals[i].id, "was", allIntervals[i].activity)
						//sendDBSchan[allIntervals[i].id] <- allIntervals[i].activity
					}
				} else if allIntervals[i].incycle {
					allIntervals[i].incycle = false
					//fmt.Println("exit cycle")
					//fmt.Println("space activity for interval", allIntervals[i].id, "was", allIntervals[i].activity)
					sendDBSchan[allIntervals[i].id] <- allIntervals[i].activity
					allIntervals[i].activity.val = 0
					allIntervals[i].activity.ts = support.Timestamp()
				}
			}
		}
	}

}
