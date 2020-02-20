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
	//if _, e := storage.SetSeries(label, v.interval, !support.StringEnding(label, "current", "_")); e != nil {
	//	log.Fatal("spaces.setUpDataDBSBank: fatal error setting database %v:%v\n", name+v.name, v.interval)
	//}
	//go dt.cf(dl+name+v.name, latestDBSIn[dl][name][v.name], nil)
	var copyAllIntervals []IntervalDetector
	timeoutInterval := 5 * chanTimeout * time.Millisecond
	active := true
	// saved := false
	// load the configuration, this is done only once even when recovering
	once.Do(func() {
		// configuration is read, if name is truncated, the name needs to be recovered from the config file as well
		var spaceName string
		if name[len(name)-1] != '_' {
			// we need to recovery the name since truncated
			sps := strings.Split(strings.Trim(os.Getenv("SPACES_NAMES"), " "), " ")
			for _, nm := range sps {
				nmck := strings.Trim(strings.Split(nm, ":")[0], " ")
				if name == nmck[0:len(name)] {
					spaceName = nmck
				}
			}
		} else {
			spaceName = strings.Trim(name, "_")
		}
		if sts := os.Getenv("PRESENCE_" + spaceName); sts != "" {
			// all intervals are read
			for _, st := range strings.Split(strings.Trim(sts, " "), ";") {
				stdata := strings.Split(strings.Trim(st, " "), " ")
				if start, e := time.Parse(support.TimeLayout, strings.Trim(stdata[1], " ")); e == nil {
					if end, e := time.Parse(support.TimeLayout, strings.Trim(stdata[2], " ")); e == nil {
						spaceName = support.StringLimit(spaceName, support.LabelLength)
						nm := support.StringLimit("presence", support.LabelLength) + spaceName + support.StringLimit(strings.Trim(stdata[0], " "), support.LabelLength)
						allIntervals = append(allIntervals, IntervalDetector{nm,
							start, end, false, DataEntry{id: nm}})
						sendDBSchan[nm] = make(chan interface{})
						//label := spaceName + nm
						//fmt.Println(nm, int(end.Unix() - start.Unix())/4, SamplingWindow)
						//os.Exit(1)
						// this approach is quite slow
						//if _, e := storage.SetSeries(nm, SamplingWindow*10, true); e != nil {
						if _, e := storage.SetSeries(nm, 0, true); e != nil {
							log.Fatalf("spaces.detectors: fatal error setting database %v\n", nm)
						}
						go dataTypes["presence"].cf(nm, sendDBSchan[nm], nil)
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
					if data[0] == spaceName {
						// select only the data for this space
						// [livlab__ presencelivlab__morning_ -62167190400 -62167183200 0 0]
						//fmt.Println(spaceName, data)
						if st, err := strconv.ParseInt(data[2], 10, 64); err == nil {
							if en, err := strconv.ParseInt(data[3], 10, 64); err == nil {
								if ts, err := strconv.ParseInt(data[4], 10, 64); err == nil {
									if val, err := strconv.Atoi(data[5]); err == nil {
										if (support.Timestamp() - ts*1000) <= CrashMaxDelay {
											inc, _ := support.InClosureTime(time.Unix(st, 0), time.Unix(en, 0))
											recData[data[1]] = IntervalDetector{Id: data[1], Start: time.Unix(st, 0), End: time.Unix(en, 0),
												inCycle: inc, Activity: DataEntry{Ts: ts, id: data[1], NetFlow: val}}
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
						log.Printf("spaces.detectors: recovered presence definition and value for %v:%v\n", spaceName, val.Id)
						//we need to check is the sample ts is relevant
						//found, err := support.InClosureTimeFull(netFlow.Start, netFlow.End, time.Unix(el.Activity.Ts, 0))
						//fmt.Println(netFlow.Id, found, err)
					}
				}
			}
			log.Printf("spaces.detectors: detectors for %v activated\n", spaceName)
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
			//fmt.Println("got", sp.netFlow)
		case <-time.After(timeoutInterval):
		}
		if active {
			// check for proper Activity falling in an interval and save at the End of the interval
			// we have some Activity
			for i := range allIntervals {
				copy(copyAllIntervals, allIntervals)
				//fmt.Println("checking", allIntervals[i].Id)
				if found, e := support.InClosureTime(allIntervals[i].Start, allIntervals[i].End); e == nil && found {
					allIntervals[i].inCycle = true
					if sp.netFlow != 0 {
						allIntervals[i].Activity.NetFlow += 1
						// allIntervals[i].Activity.Ts = support.Timestamp()
						if support.Debug != 0 {
							fmt.Println("space Activity for interval", allIntervals[i].Id, "was", allIntervals[i].Activity)
						}
						// 2 activities is the minimum for guaranteed presence and we store it as soon as it happens
						//var recSH string
						//if allIntervals[i].End.Hour() >= allIntervals[i].Start.Hour() {
						//	recSH = "0" + strconv.Itoa((allIntervals[i].End.Hour()-allIntervals[i].Start.Hour())/4+allIntervals[i].Start.Hour())
						//} else {
						//	recSH = "0" + strconv.Itoa((24+allIntervals[i].End.Hour()-allIntervals[i].Start.Hour())/4+allIntervals[i].Start.Hour())
						//
						//}
						//recSM := "0" + strconv.Itoa((allIntervals[i].End.Minute()-allIntervals[i].Start.Minute())/4+allIntervals[i].Start.Minute())
						//recSH = recSH[len(recSH)-2:]
						//recSM = recSM[len(recSM)-2:]
						//fmt.Println(recSH, recSM, recSH+":"+recSM)
						//if recStart, e := time.Parse(support.TimeLayout, recSH+":"+recSM); e == nil {
						//fmt.Print(recStart)
						//if found, e := support.InClosureTime(recStart, allIntervals[i].End); e == nil && found {
						// if allIntervals[i].Activity.NetFlow >= minTransactionsForDetection && !saved {
						// 	sendDBSchan[allIntervals[i].Id] <- allIntervals[i].Activity
						// 	saved = true
						// 	//fmt.Println("space Activity for interval", allIntervals[i].Id, "saved")
						// } else {
						// 	//fmt.Println("space Activity for interval", allIntervals[i].Id, "NOT saved")
						// }
						//}
						//}
						//os.Exit(1)
					}
				} else if allIntervals[i].inCycle {
					// sample is saved with a ts adjusted with the timeout
					allIntervals[i].Activity.Ts = support.Timestamp() - 5*chanTimeout*2
					if support.Debug != 0 {
						fmt.Println("space Activity for interval", allIntervals[i].Id, " ended as", allIntervals[i].Activity)
					}
					allIntervals[i].inCycle = false
					// saved = false
					//fmt.Println("exit cycle")
					//fmt.Println("space Activity for interval", allIntervals[i].Id, "was", allIntervals[i].Activity)
					sendDBSchan[allIntervals[i].Id] <- allIntervals[i].Activity
					allIntervals[i].Activity.NetFlow = 0
				} else {
					//fmt.Println("not cycle")
				}
			}
			// send the current values to the recovery register
			recovery <- copyAllIntervals
		}
	}

}
