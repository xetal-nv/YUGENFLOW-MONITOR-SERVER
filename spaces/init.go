package spaces

import (
	"bufio"
	"gateserver/storage"
	"gateserver/support"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// set-up of all space threads, channels and variables based on configuration file .env
func SetUp() {
	// set-up the data types and conversions
	dtypes = make(map[string]dtfuncs)
	var sample = dtfuncs{}
	sample.pf = func(nm string, se spaceEntries) interface{} {
		return dataEntry{id: nm, ts: se.ts, val: se.val}
	}
	sample.cf = func(id string, in chan interface{}, rst chan bool) {
		storage.SerieSampleDBS(id, in, rst)
	}
	dtypes[support.StringLimit("sample", support.LabelLength)] = sample
	var entry = dtfuncs{}
	entry.pf = func(nm string, se spaceEntries) interface{} {
		data := struct {
			id      string
			ts      int64
			length  int
			entries [][]int
		}{ts: 0}
		if len(se.entries) > 0 {
			var entries [][]int
			for id, v := range se.entries {
				entries = append(entries, []int{id, v.val})
			}
			data.id = nm
			data.ts = se.ts
			data.length = len(entries)
			data.entries = entries
		}
		return data
	}
	entry.cf = func(id string, in chan interface{}, rst chan bool) {
		storage.SeriesEntryDBS(id, in, rst)
	}
	dtypes[support.StringLimit("entry", support.LabelLength)] = entry

	// set multicycleonlydays
	if data := os.Getenv("MULTICYCLEDAYSONLY"); data != "" {
		multicycleonlydays = true
		log.Printf("spaces.setUpDataDBSBank: MULTICYCLEDAYSONLY enabled\n")
	} else {
		multicycleonlydays = false
	}
	// load initial values is present (note maximum line size 64k characters)
	MutexInitData.Lock()
	if data := os.Getenv("SPACES_NAMES"); data != "" {
		spaces := strings.Split(data, " ")

		InitData = make(map[string]map[string]map[string][]string)
		for i := range dtypes {
			InitData[i] = make(map[string]map[string][]string)
			for _, j := range spaces {
				name := strings.Trim(strings.Split(j, ":")[0], " ")
				InitData[i][support.StringLimit(name, support.LabelLength)] = make(map[string][]string)
			}
		}
		//fmt.Println(InitData)
		file, err := os.Open(".recovery")
		if err != nil {
			log.Printf("spaces.setUpDataDBSBank: InitData file absent or corrupted\n")
		} else {
			//noinspection GoUnhandledErrorResult
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				data := strings.Split(scanner.Text(), ",")
				//fmt.Println(data)
				if len(data) == 5 {
					// any other length implies a wrong data and will be ignored
					InitData[data[0]][data[1]][data[2]] = []string{data[3], data[4]}
				}
			}
			//fmt.Println(InitData)
			log.Printf("spaces.setUpDataDBSBank: InitData file imported\n")
		}
		//fmt.Println(InitData)
		//os.Exit(1)
	} else {
		log.Printf("spaces.setUpDataDBSBank: error reading space definition, recovery skipped")
	}
	MutexInitData.Unlock()

	// set up the server for data analysis/transmission/storage
	setpUpCounter()
	spchans := setUpSpaces()
	setUpDataDBSBank(spchans)
}

// set-up space thread and data flow structure based on the provided configuration
func setUpSpaces() (spaceChannels map[string]chan spaceEntries) {
	spaceChannels = make(map[string]chan spaceEntries)
	entrySpaceChannels = make(map[int][]chan spaceEntries)
	SpaceDef = make(map[string][]int)
	spaceTimes = make(map[string]timeSchedule)

	if data := os.Getenv("SPACES_NAMES"); data != "" {
		SpaceMaxOccupancy = make(map[string]int)
		spaces := strings.Split(data, " ")
		bufsize = 50
		if v, e := strconv.Atoi(os.Getenv("INTBUFFSIZE")); e == nil {
			bufsize = v
		}

		// extracts the paces names and their maximum occupancy is defined
		for i := 0; i < len(spaces); i++ {
			singleSpaceData := strings.Split(strings.Trim(spaces[i], " "), ":")
			name := strings.Trim(singleSpaceData[0], " ")
			spaces[i] = strings.Trim(name, " ")
			if len(singleSpaceData) == 2 {
				if v, e := strconv.Atoi(singleSpaceData[1]); e == nil {
					SpaceMaxOccupancy[support.StringLimit(spaces[i], support.LabelLength)] = v
				} else {
					log.Fatal("spaces.setUpSpaces: fatal error entry maximum occupancy", singleSpaceData)
				}
			}
		}

		//fmt.Println(spaces)
		//fmt.Println(SpaceMaxOccupancy)
		//os.Exit(1)

		// initialise the processing threads for each space
		for _, name := range spaces {
			var sprange timeSchedule
			nl, _ := time.Parse(support.TimeLayout, "00:00")
			rng := strings.Split(strings.Trim(os.Getenv("CLOSURE_"+name), ";"), ";")
			sprange.start, sprange.end = nl, nl
			if len(rng) == 2 {
				for i, v := range rng {
					rng[i] = strings.Trim(v, " ")
				}
				if v, e := time.Parse(support.TimeLayout, rng[0]); e == nil {
					sprange.start = v
					if v, e := time.Parse(support.TimeLayout, rng[1]); e == nil {
						sprange.end = v
					} else {
						sprange.start = nl
					}
				}
			}
			spaceTimes[support.StringLimit(name, support.LabelLength)] = sprange
			if sts := os.Getenv("SPACE_" + name); sts != "" {
				name = support.StringLimit(name, support.LabelLength)
				spaceChannels[name] = make(chan spaceEntries, bufsize)
				// the go routine below is the processing thread.
				go sampler(name, spaceChannels[name], nil, nil, nil, 0, sync.Once{}, 0, 0)
				var sg []int
				for _, val := range strings.Split(sts, " ") {
					vt := strings.Trim(val, " ")
					if v, e := strconv.Atoi(vt); e == nil {
						sg = append(sg, v)
					} else {
						log.Fatal("spaces.setUpSpaces: fatal error entry name", val)
					}
				}
				SpaceDef[name] = sg
				log.Printf("spaces.setUpSpaces: found space [%v] with entry %v\n", name, sg)
				for _, g := range sg {
					entrySpaceChannels[g] = append(entrySpaceChannels[g], spaceChannels[name])
				}
			} else {
				log.Printf("spaces.setUpSpaces: found  empty space [%v]\n", name)
			}
		}
	} else {
		log.Fatal("spaces.setUpSpaces: fatal error no space has been defined")
	}
	//time.Sleep(5 * time.Second)
	//os.Exit(1)
	return spaceChannels
}

// set-up counters thread and data flow structure based on the provided configuration
func setpUpCounter() {
	cmode = os.Getenv("CMODE")
	if cmode == "" {
		cmode = "0"
	}
	log.Printf("Compression mode set to %v\n", cmode)

	//cstats = os.Getenv("CSTATS")
	//if cstats == "" {
	//	cstats = "0"
	//}
	//if cstats == "1" {
	//	log.Printf("Compression mode enabled for statistics\n")
	//}

	//sw := os.Getenv("SAMWINDOW")
	if os.Getenv("INSTANTNEG") == "1" {
		instNegSkip = false
	} else {
		instNegSkip = true
	}
	log.Printf("spaces.setpUpCounter: setting flag for skipping negative samples to %v\n", instNegSkip)
	if os.Getenv("AVERAGENEG") == "1" {
		avgNegSkip = false
	} else {
		avgNegSkip = true
	}
	log.Printf("spaces.setpUpCounter: setting flag for skipping negative averages to %v\n", avgNegSkip)

	if sw := os.Getenv("SAMWINDOW"); sw == "" {
		SamplingWindow = 30
	} else {
		if v, e := strconv.Atoi(sw); e != nil {
			log.Fatal("spaces.setpUpCounter: fatal error in definition of SAMWINDOW")
		} else {
			SamplingWindow = v
			//}
		}
	}
	log.Printf("spaces.setpUpCounter: setting sliding window at %vs\n", SamplingWindow)

	avgw := strings.Trim(os.Getenv("ANALYSISPERIOD"), ";")
	avgWindows := make(map[string]int)
	//avgAnalysisSchedule = make(map[string]timeSchedule)
	tw := make(map[int]string)
	curr := support.StringLimit("current", support.LabelLength)
	avgWindows[curr] = SamplingWindow
	tw[SamplingWindow] = curr
	if avgw != "" {
		for _, v := range strings.Split(avgw, ";") {
			data := strings.Split(strings.Trim(v, " "), " ")

			// switch is used instead of if statement for future extensions
			switch len(data) {
			case 2:
				// analysis defined with period only, nothing extra to be done
			//case 4:
			// analysis defined with start and end
			//if st, e := time.Parse(support.TimeLayout, data[2]); e == nil {
			//	//fmt.Println("start", st)
			//	if en, e := time.Parse(support.TimeLayout, data[3]); e == nil {
			//		//fmt.Println("end", en)
			//		avgAnalysisSchedule[support.StringLimit(data[0], support.LabelLength)] = timeSchedule{st, en}
			//	} else {
			//		log.Println("spaces.setpUpCounter: illegal end ANALYSISPERIOD value", data)
			//	}
			//} else {
			//	log.Println("spaces.setpUpCounter: illegal start ANALYSISPERIOD value", data)
			//}
			default:
				// error
				log.Fatal("spaces.setpUpCounter: fatal error for illegal ANALYSISPERIOD values", data)
			}

			if v, e := strconv.Atoi(data[1]); e != nil {
				log.Fatal("spaces.setpUpCounter: fatal error for illegal ANALYSISPERIOD values", data)
			} else {
				if v > SamplingWindow {
					name := support.StringLimit(data[0], support.LabelLength)
					avgWindows[name] = v
					tw[v] = name
				} else {
					log.Printf("spaces.setpUpCounter: averaging window %v skipped since equal to \"current\"\n", data[0])
				}
			}
		}
	}

	keys := make([]int, 0, len(tw))
	avgAnalysis = make([]avgInterval, len(tw))
	for k := range tw {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for i, v := range keys {
		avgAnalysis[i] = avgInterval{tw[v], v}
	}
	log.Printf("spaces.setpUpCounter: setting averaging windows at \n  %v\n", avgAnalysis)
	//if len(avgAnalysisSchedule) > 0 {
	//	tmp := ""
	//	for i := range avgAnalysisSchedule {
	//		tmp += i
	//	}
	//	log.Printf("spaces.setpUpCounter: setting averaging time schedule for \n  [%v]\n", tmp)
	//}

	//fmt.Println(avgAnalysis)
	//fmt.Println(avgAnalysisSchedule)

	//os.Exit(1)

	jsTxt := "var openingTime = \"\";\n"

	if val := strings.Split(strings.Trim(os.Getenv("ANALYSISWINDOW"), " "), " "); len(val) == 2 {
		if st, e := time.Parse(support.TimeLayout, val[0]); e == nil {
			//fmt.Println("start", st)
			if en, e := time.Parse(support.TimeLayout, val[1]); e == nil {
				//fmt.Println("end", en)
				jsTxt = "var openingTime = \"from " + val[0] + " to " + val[1] + "\";\n"
				avgAnalysisSchedule = timeSchedule{st, en, 0}
				avgAnalysisSchedule.duration, _ = support.TimeDifferenceInSecs(val[0], val[1])
				avgAnalysisSchedule.duration += 60000
				//fmt.Println(avgAnalysisSchedule)
				log.Printf("spaces.setpUpCounter: Analysis window is set from %v to %v\n", val[0], val[1])
			} else {
				log.Fatal("spaces.setpUpCounter: illegal end ANALYSISWINDOW value", val)
			}
		} else {
			log.Fatal("spaces.setpUpCounter: illegal start ANALYSISWINDOW value", val)
		}
	}

	f, err := os.Create("./html/js/op.js")
	if err != nil {
		log.Fatal("Fatal error creating op.js: ", err)
	}
	if _, err := f.WriteString(jsTxt); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to op.js: ", err)
	}
	if err = f.Close(); err != nil {
		log.Fatal("Fatal error closing op.js: ", err)
	}
	//os.Exit(1)
}

// set-up DBS thread and data flow structure based on the provided configuration
func setUpDataDBSBank(spaceChannels map[string]chan spaceEntries) {

	now := support.Timestamp()

	//fmt.Println(InitData)

	//os.Exit(1)

	latestChannelLock.Lock()
	LatestBankOut = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	latestBankIn = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	latestDBSIn = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	ResetDBS = make(map[string]map[string]map[string]chan bool, len(spaceChannels))

	for dl, dt := range dtypes {

		LatestBankOut[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))
		latestBankIn[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))

		if support.Debug < 3 {
			latestDBSIn[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))
			ResetDBS[dl] = make(map[string]map[string]chan bool, len(spaceChannels))
		}

		for name := range spaceChannels {
			LatestBankOut[dl][name] = make(map[string]chan interface{}, len(avgAnalysis))
			latestBankIn[dl][name] = make(map[string]chan interface{}, len(avgAnalysis))
			if support.Debug < 3 {
				ResetDBS[dl][name] = make(map[string]chan bool, len(avgAnalysis))
				latestDBSIn[dl][name] = make(map[string]chan interface{}, len(avgAnalysis))
			}
			for _, v := range avgAnalysis {
				LatestBankOut[dl][name][v.name] = make(chan interface{})
				latestBankIn[dl][name][v.name] = make(chan interface{})
				// formatting of initialisation data
				if len(InitData[dl][name][v.name]) == 2 {
					// possibly valid InitData data found
					tag := dl + name + v.name
					if ts, err := strconv.ParseInt(InitData[dl][name][v.name][0], 10, 64); err == nil {
						// found valid InitData data
						if (now - ts) < Crashmaxdelay {
							// data is fresh enough
							//fmt.Println("accepted", now-ts, InitData)
							switch dl {
							case "sample__":
								if va, e := strconv.Atoi(InitData[dl][name][v.name][1]); e == nil {
									//fmt.Println("e ", dataEntry{tag, ts, va})
									go storage.SafeReg(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name], dataEntry{tag, ts, va})
								} else {
									log.Printf("spaces.setUpDataDBSBank: invalid InitData data for %v\n", tag)
									//fmt.Println(dl + name + v.name, "starts with no init")
									go storage.SafeReg(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
								}
							case "entry___":
								vas := strings.Split(InitData[dl][name][v.name][1][2:len(InitData[dl][name][v.name][1])-2], "][")
								var va [][]int
								for _, el := range vas {
									sd := strings.Split(el, " ")
									if len(sd) == 2 {
										sd0, e0 := strconv.Atoi(sd[0])
										sd1, e1 := strconv.Atoi(sd[1])
										if e0 == nil && e1 == nil {
											va = append(va, []int{sd0, sd1})
										} else {
											log.Printf("spaces.setUpDataDBSBank: invalid InitData data for %v\n", tag)
										}
									} else {
										log.Printf("spaces.setUpDataDBSBank: invalid InitData data for %v\n", tag)
									}
								}
								if len(va) > 0 {
									data := struct {
										id      string
										ts      int64
										length  int
										entries [][]int
									}{id: tag, ts: 0, length: len(va), entries: va}
									//fmt.Println("e ", tag, ts, va)
									go storage.SafeReg(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name], data)
								} else {
									log.Printf("spaces.setUpDataDBSBank: invalid InitData data for %v\n", tag)
									//fmt.Println(dl + name + v.name, "starts with no init")
									go storage.SafeReg(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
								}
							default:
								log.Printf("spaces.setUpDataDBSBank: invalid InitData data type for %v\n", tag)
								//fmt.Println(dl + name + v.name, "starts with no init")
								go storage.SafeReg(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
							}
						} else {
							// data is not fresh enough
							//fmt.Println(now, ts, Crashmaxdelay)
							log.Printf("spaces.setUpDataDBSBank: too old InitData ts for %v\n", tag)
							//fmt.Println(dl + name + v.name, "starts with no init since InitData is old")
							go storage.SafeReg(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
						}
					} else {
						log.Printf("spaces.setUpDataDBSBank: invalid InitData ts for %v\n", tag)
						//fmt.Println(dl + name + v.name, "starts with no init")
						go storage.SafeReg(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
					}
				} else {
					// register started with no InitData data (not available)
					//fmt.Println(dl + name + v.name, "starts with no init")
					go storage.SafeReg(dl+name+v.name, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
				}

				// start of distributed data passing structure
				//go storage.SafeReg(latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
				if support.Debug < 3 {
					ResetDBS[dl][name][v.name] = make(chan bool)
					latestDBSIn[dl][name][v.name] = make(chan interface{})
					label := dl + name + v.name
					if _, e := storage.SetSeries(label, v.interval, !support.Stringending(label, "current", "_")); e != nil {
						log.Fatal("spaces.setUpDataDBSBank: fatal error setting database %v:%v\n", name+v.name, v.interval)
					}
					go dt.cf(dl+name+v.name, latestDBSIn[dl][name][v.name], ResetDBS[dl][name][v.name])
				}
			}
			log.Printf("spaces.setUpDataDBSBank: DataBank for space %v and data %v initialised\n", name, dl)
		}
	}
	latestChannelLock.Unlock()
}
