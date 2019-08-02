package spaces

import (
	"bufio"
	"fmt"
	"gateserver/storage"
	"gateserver/support"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TODO add recovery values form crashes from .recoverypres

// SafeRegDetectors is a safe register used for recovery purposes
func SafeRegDetectors(in, out chan []IntervalDetector) {
	var data []IntervalDetector
	r := func() {
		for {
			select {
			case data = <-in:
			case out <- data:
			}
		}
	}
	go support.RunWithRecovery(r, nil)
}

//func detectors(spacename string, prevStageChan, nextStageChan chan spaceEntries, syncPrevious, syncNext chan bool, avgID int, once sync.Once, tn, ntn int) {
func detectors(name string, gateChan chan spaceEntries, allIntervals []IntervalDetector, recovery chan []IntervalDetector,
	sendDBSchan map[string]chan interface{}, once sync.Once, tn, ntn int) {

	//latestDBSIn[dl][name][v.name] = make(chan interface{})
	//label := dl + name + v.name
	//if _, e := storage.SetSeries(label, v.interval, !support.Stringending(label, "current", "_")); e != nil {
	//	log.Fatal("spaces.setUpDataDBSBank: fatal error setting database %v:%v\n", name+v.name, v.interval)
	//}
	//go dt.cf(dl+name+v.name, latestDBSIn[dl][name][v.name], nil)

	timeoutInterval := 5 * chantimeout * time.Millisecond
	active := true
	// load the configuration, this is done only once even when recovering
	once.Do(func() {
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
						allIntervals = append(allIntervals, IntervalDetector{nm,
							start, end, false, DataEntry{id: nm}})
						sendDBSchan[nm] = make(chan interface{})
						//label := spacename + nm
						if _, e := storage.SetSeries(nm, SamplingWindow, true); e != nil {
							log.Fatalf("spaces.detectors: fatal error setting database %v\n", nm)
						}
						go dtypes["presence"].cf(nm, sendDBSchan[nm], nil)
					}
				}
			}

			// load recovery data
			file, err := os.Open(".recoverypres")
			if err != nil {
				log.Printf("spaces.detectors: recoverypres file absent or corrupted\n")
			} else {
				//noinspection GoUnhandledErrorResult
				defer file.Close()
				scanner := bufio.NewScanner(file)
				recData := make(map[string]IntervalDetector)
				for scanner.Scan() {
					data := strings.Split(scanner.Text(), ",")
					if data[0] == spacename {
						// select only the data for this space
						// [livlab__ presencelivlab__morning_ -62167190400 -62167183200 0 0]
						//fmt.Println(spacename, data)
						if st, err := strconv.ParseInt(data[2], 10, 64); err == nil {
							if en, err := strconv.ParseInt(data[3], 10, 64); err == nil {
								if ts, err := strconv.ParseInt(data[4], 10, 64); err == nil {
									if val, err := strconv.Atoi(data[5]); err == nil {
										if (support.Timestamp() - ts*1000) <= Crashmaxdelay {
											inc, _ := support.InClosureTime(time.Unix(st, 0), time.Unix(en, 0))
											recData[data[1]] = IntervalDetector{Id: data[1], Start: time.Unix(st, 0), End: time.Unix(en, 0),
												incycle: inc, Activity: DataEntry{Ts: ts, id: data[1], Val: val}}
										}
									}
								}
							}
						}
					}
				}
				//for _, v := range allIntervals {
				//	fmt.Println(v)
				//}
				//for _, v := range recData {
				//	fmt.Println(v)
				//}
				for i, val := range allIntervals {
					if el, ok := recData[val.Id]; ok {
						allIntervals[i] = el
						log.Printf("spaces.detectors: recovered presence definition and value for %v:%v\n", spacename, val.Id)
						//we need to check is the sample ts is relevant
						//found, err := support.InClosureTimeFull(val.Start, val.End, time.Unix(el.Activity.Ts, 0))
						//fmt.Println(val.Id, found, err)
					}
				}
			}

		} else {
			active = false
			//fmt.Println("detector for", name, "not active")
		}
	})

	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"spaces.detector: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			log.Printf("spaces.detectors: space %v detector recovering from : %v\n ", name, e)
			go detectors(name, gateChan, allIntervals, recovery, sendDBSchan, once, tn, ntn)
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
			// check for proper Activity falling in an interval and save at the End of the interval
			// we have some Activity
			for i := range allIntervals {
				//fmt.Println("checking", allIntervals[i].Id)
				if found, e := support.InClosureTime(allIntervals[i].Start, allIntervals[i].End); e == nil && found {
					allIntervals[i].incycle = true
					if sp.val != 0 {
						allIntervals[i].Activity.Val += 1
						allIntervals[i].Activity.Ts = support.Timestamp()
						if support.Debug != 0 {
							fmt.Println("space Activity for interval", allIntervals[i].Id, "was", allIntervals[i].Activity)
						}
						//sendDBSchan[allIntervals[i].Id] <- allIntervals[i].Activity
					}
				} else if allIntervals[i].incycle {
					if support.Debug != 0 {
						fmt.Println("space Activity for interval", allIntervals[i].Id, " ended as", allIntervals[i].Activity)
					}
					allIntervals[i].incycle = false
					//fmt.Println("exit cycle")
					//fmt.Println("space Activity for interval", allIntervals[i].Id, "was", allIntervals[i].Activity)
					sendDBSchan[allIntervals[i].Id] <- allIntervals[i].Activity
					allIntervals[i].Activity.Val = 0
					allIntervals[i].Activity.Ts = support.Timestamp()
				}
			}
			// send the current values to the recovery register
			recovery <- allIntervals
		}
	}

}
