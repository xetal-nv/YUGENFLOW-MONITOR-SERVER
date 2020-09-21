package spaces

import (
	"bufio"
	"fmt"
	"gateserver/storage"
	"gateserver/supp"
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
	dataTypes = make(map[string]dtFunctions)

	// all possible data to be stored and distributed needs to have an entry in dataTypes
	// the data type needs to implement the following interfaces:
	//  storage.sampledata, servers.genericdata
	// currently defined are: sample, entry, presence
	// track usage of dataTypes and use proper conditional statements

	// add sample type based on DataEntry
	var sample = dtFunctions{}
	sample.pf = func(nm string, se spaceEntries) interface{} {
		return DataEntry{id: nm, Ts: se.ts, NetFlow: se.netFlow}
	}
	sample.cf = func(id string, in chan interface{}, rst chan bool) {
		storage.SerieSampleDBS(id, in, rst, "TS")
	}
	dataTypes[supp.StringLimit("sample", supp.LabelLength)] = sample
	// add entry type
	var entry = dtFunctions{}
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
				entries = append(entries, []int{id, v.NetFlow, v.PositiveFlow, v.NegativeFlow})
			}
			data.id = nm
			data.ts = se.ts
			data.length = len(entries)
			data.entries = entries
		}
		return data
	}
	entry.cf = func(id string, in chan interface{}, rst chan bool) {
		storage.SeriesEntryDBS(id, in, rst, "TS")
	}
	dataTypes[supp.StringLimit("entry", supp.LabelLength)] = entry
	// add presence type, which is equal to sample
	var presence = dtFunctions{}
	presence.pf = func(nm string, se spaceEntries) interface{} {
		return DataEntry{id: nm, Ts: se.ts, NetFlow: se.netFlow}
	}
	presence.cf = func(id string, in chan interface{}, rst chan bool) {
		storage.SerieSampleDBS(id, in, rst, "SD")
	}
	dataTypes[supp.StringLimit("presence", supp.LabelLength)] = presence

	// set multiCycleOnlyDays
	if data := os.Getenv("MULTICYCLEDAYSONLY"); data != "" {
		multiCycleOnlyDays = true
		log.Printf("spaces.setUpDataDBSBank: MULTICYCLEDAYSONLY enabled\n")
	} else {
		multiCycleOnlyDays = false
	}
	// load initial values is present (note maximum line size 64k characters)
	MutexInitData.Lock()
	if data := os.Getenv("SPACES_NAMES"); data != "" {
		spaces := strings.Split(data, " ")

		InitData = make(map[string]map[string]map[string][]string)
		for i := range dataTypes {
			// filter out data that do not need Start value form recovery
			switch i {
			case "presence":
				// skip
			default:
				// define
				InitData[i] = make(map[string]map[string][]string)
				for _, j := range spaces {
					name := strings.Trim(strings.Split(j, ":")[0], " ")
					InitData[i][supp.StringLimit(name, supp.LabelLength)] = make(map[string][]string)
				}
			}
		}

		file, err := os.Open(".recoveryavg")
		if err != nil {
			log.Printf("spaces.setUpDataDBSBank: InitData file absent or corrupted\n")
		} else {
			//noinspection GoUnhandledErrorResult
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				//fmt.Println(scanner.Text())
				data := strings.Split(scanner.Text(), ",")
				// any other length implies a wrong data and will be ignored
				if len(data) == 5 {
					// this is a redundant check
					if InitData[data[0]] == nil {
						fmt.Println("spaces.setUpDataDBSBank: fatal error reading initial values")
						os.Exit(1)
					}
					if InitData[data[0]][data[1]] == nil {
						fmt.Println("spaces.setUpDataDBSBank: fatal error reading initial values")
						os.Exit(1)
					}
					InitData[data[0]][data[1]][data[2]] = []string{data[3], data[4]}
				}
			}
			//fmt.Println(InitData)
			log.Printf("spaces.setUpDataDBSBank: InitData file imported\n")
		}
	} else {
		log.Printf("spaces.setUpDataDBSBank: error reading space definition, recovery skipped")
	}
	MutexInitData.Unlock()

	// set up the server for data analysis/transmission/storage
	setpUpCounter()
	spChans := setUpSpaces()
	setUpDataDBSBank(spChans)

	// shadowAnalysis is set in setpUpCounter
	shadowSingleMux.Lock()
	if shadowAnalysis != "" {
		for i := range SpaceDef {
			var e error
			_, err := os.Stat("log/shadowreport_" + strings.Trim(i, "_") + "_" + strings.Trim(shadowAnalysis, "_") + ".txt")
			if shadowAnalysisFile[i], e = os.OpenFile("log/shadowreport_"+strings.Trim(i, "_")+"_"+strings.Trim(shadowAnalysis, "_")+".txt",
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); e != nil {
				log.Println("spaces.SetUp: fatal error unable to create or open shadow file for space ", i) // redundant will be removed later
				log.Fatal("spaces.SetUp: fatal error unable to create or open shadow file for space ", i)
			} else {
				shadowAnalysisDate[i] = ""
				log.Println("spaces.SetUp: created or opened shadow file for space ", i)
				if !os.IsNotExist(err) {
					_, _ = shadowAnalysisFile[i].WriteString("\n")
				}
			}
		}
	}
	shadowSingleMux.Unlock()
}

// set-up space thread and data flow structure based on the provided configuration
func setUpSpaces() (spaceChannels map[string]chan spaceEntries) {
	spaceChannels = make(map[string]chan spaceEntries)
	spacePresenceChannels := make(map[string]chan spaceEntries)
	entrySpaceSamplerChannels = make(map[int][]chan spaceEntries)
	entrySpacePresenceChannels = make(map[int][]chan spaceEntries)
	SpaceDef = make(map[string][]int)
	SpaceTimes = make(map[string]TimeSchedule)

	if data := os.Getenv("SPACES_NAMES"); data != "" {
		SpaceMaxOccupancy = make(map[string]int)
		spaces := strings.Split(data, " ")
		bufferSize = 50
		if v, e := strconv.Atoi(os.Getenv("INTBUFFSIZE")); e == nil {
			bufferSize = v
		}

		// extracts the paces names and their maximum occupancy is defined
		for i := 0; i < len(spaces); i++ {
			singleSpaceData := strings.Split(strings.Trim(spaces[i], " "), ":")
			name := strings.Trim(singleSpaceData[0], " ")
			spaces[i] = strings.Trim(name, " ")
			if len(singleSpaceData) == 2 {
				if v, e := strconv.Atoi(singleSpaceData[1]); e == nil {
					SpaceMaxOccupancy[supp.StringLimit(spaces[i], supp.LabelLength)] = v
				} else {
					log.Fatal("spaces.setUpSpaces: fatal error entry maximum occupancy", singleSpaceData)
				}
			}
		}

		// initialise the processing threads for each space
		for _, name := range spaces {
			var spRange TimeSchedule
			nl, _ := time.Parse(supp.TimeLayout, "00:00")
			rng := strings.Split(strings.Trim(os.Getenv("CLOSURE_"+name), ";"), ";")
			spRange.Start, spRange.End = nl, nl
			if len(rng) == 2 {
				for i, v := range rng {
					rng[i] = strings.Trim(v, " ")
				}
				if v, e := time.Parse(supp.TimeLayout, rng[0]); e == nil {
					spRange.Start = v
					if v, e := time.Parse(supp.TimeLayout, rng[1]); e == nil {
						spRange.End = v
					} else {
						spRange.Start = nl
					}
				}
			}
			SpaceTimes[supp.StringLimit(name, supp.LabelLength)] = spRange
			LatestDetectorOut = make(map[string]chan []IntervalDetector)
			if sts := os.Getenv("SPACE_" + name); sts != "" {
				name = supp.StringLimit(name, supp.LabelLength)

				// the go routines below Start the relevant processing threads
				// sampling and averaging threads
				spaceChannels[name] = make(chan spaceEntries, bufferSize)
				go sampler(name, spaceChannels[name], nil, nil, nil, 0, 0, 0, spaceEntries{invalid: true})

				// presence detection threads
				spacePresenceChannels[name] = make(chan spaceEntries, bufferSize)
				LatestDetectorOut[name] = make(chan []IntervalDetector)
				detectCh := make(chan []IntervalDetector)
				go SafeRegDetectors(detectCh, LatestDetectorOut[name])
				go detectors(name, spacePresenceChannels[name], []IntervalDetector{}, detectCh,
					make(map[string]chan interface{}), sync.Once{}, 0, 0)

				var sg []int
				for _, val := range strings.Split(sts, " ") {
					vt := strings.Trim(val, " ")
					if v, e := strconv.Atoi(vt); e == nil {
						sg = append(sg, v)
					} else {
						log.Fatal("spaces.setUpSpaces: fatal error entry name", val)
					}
				}
				sort.Ints(sg)
				SpaceDef[name] = sg
				log.Printf("spaces.setUpSpaces: found space [%v] with entry %v\n", name, sg)
				for _, g := range sg {
					entrySpaceSamplerChannels[g] = append(entrySpaceSamplerChannels[g], spaceChannels[name])
					entrySpacePresenceChannels[g] = append(entrySpacePresenceChannels[g], spacePresenceChannels[name])
				}
			} else {
				log.Printf("spaces.setUpSpaces: found  empty space [%v]\n", name)
			}
		}
	} else {
		log.Fatal("spaces.setUpSpaces: fatal error no space has been defined")
	}
	storage.SpaceInfo = SpaceDef
	return spaceChannels
}

// set-up counters thread and data flow structure based on the provided configuration
func setpUpCounter() {
	compressionMode = os.Getenv("CMODE")
	if compressionMode == "" {
		compressionMode = "0"
	}
	log.Printf("Compression mode set to %v\n", compressionMode)

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
		}
	}
	log.Printf("spaces.setpUpCounter: setting sliding window at %vs\n", SamplingWindow)

	avgWin := strings.Trim(os.Getenv("ANALYSISPERIOD"), ";")
	avgWindows := make(map[string]int)
	tw := make(map[int]string)
	curr := supp.StringLimit("current", supp.LabelLength)
	avgWindows[curr] = SamplingWindow
	tw[SamplingWindow] = curr
	if avgWin != "" {
		for _, v := range strings.Split(avgWin, ";") {
			data := strings.Split(strings.Trim(v, " "), " ")

			// switch is used instead of if statement for future extensions
			switch len(data) {
			case 2:
				// analysis defined with period only, nothing extra to be done
			default:
				// error
				log.Fatal("spaces.setpUpCounter: fatal error for illegal ANALYSISPERIOD values", data)
			}

			if v, e := strconv.Atoi(data[1]); e != nil {
				log.Fatal("spaces.setpUpCounter: fatal error for illegal ANALYSISPERIOD values", data)
			} else {
				if v > SamplingWindow {
					name := supp.StringLimit(data[0], supp.LabelLength)
					avgWindows[name] = v
					tw[v] = name
				} else {
					log.Printf("spaces.setpUpCounter: averaging window %v skipped since equal to \"current\"\n", data[0])
				}
			}
		}
	} else {
		log.Fatal("spaces.setpUpCounter: no averaging values have been defined in ANALYSISPERIOD!!!")
	}

	keys := make([]int, 0, len(tw))
	AvgAnalysis = make([]AvgInterval, len(tw))
	for k := range tw {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for i, v := range keys {
		AvgAnalysis[i] = AvgInterval{tw[v], v}
	}
	log.Printf("spaces.setpUpCounter: setting averaging windows at \n  %v\n", AvgAnalysis)

	shadowSingleMux.Lock()
	shadowAnalysis = strings.Trim(os.Getenv("SHADOWREPORTING"), " ")
	found := false
	for _, k := range AvgAnalysis {
		if shadowAnalysis == strings.Trim(k.Name, "_") {
			found = true
			break
		}
	}

	if shadowAnalysis != "" && found {
		shadowAnalysis = supp.StringLimit(shadowAnalysis, supp.LabelLength)
		shadowAnalysisFile = make(map[string]*os.File)
		shadowAnalysisDate = make(map[string]string)
		log.Printf("spaces.setpUpCounter: Shadow Analysis defined as (%v)\n", shadowAnalysis)
		//fmt.Printf("spaces.setpUpCounter: Shadow Analysis defined as (%v)\n", shadowAnalysis)
	} else {
		shadowAnalysis = ""
	}
	shadowSingleMux.Unlock()

	//jsTxt := "var openingTime = \"\";\n"
	//jsST := "var opStartTime = \"\";\n"
	//jsEN := "var opEndTime = \"\";\n"

	if val := strings.Split(strings.Trim(os.Getenv("ANALYSISWINDOW"), " "), " "); len(val) == 2 {
		if st, e := time.Parse(supp.TimeLayout, val[0]); e == nil {
			if en, e := time.Parse(supp.TimeLayout, val[1]); e == nil {
				//jsTxt = "var openingTime = \"from " + NetFlow[0] + " to " + NetFlow[1] + "\";\n"
				//jsST = "var opStartTime = \"" + NetFlow[0] + "\";\n"
				//jsEN = "var opEndTime = \"" + NetFlow[1] + "\";\n"
				avgAnalysisSchedule = TimeSchedule{st, en, 0}
				avgAnalysisSchedule.Duration, _ = supp.TimeDifferenceInSecs(val[0], val[1])
				avgAnalysisSchedule.Duration += 60000
				log.Printf("spaces.setpUpCounter: Analysis window is set from %v to %v\n", val[0], val[1])
			} else {
				log.Fatal("spaces.setpUpCounter: illegal End ANALYSISWINDOW value", val)
			}
		} else {
			log.Fatal("spaces.setpUpCounter: illegal Start ANALYSISWINDOW value", val)
		}
	}

	//if strings.Trim(os.Getenv("RTWINDOW"), " ") == "" {
	//
	//	//var f *os.File
	//	//var err error
	//
	//	//f, err := os.OpenFile("./html/js/def.js", os.O_APPEND|os.O_WRONLY, 0600)
	//	f, err := os.Create("./html/js/op.js")
	//	if err != nil {
	//		//f, err = os.Create("./html/js/def.js")
	//		//if err != nil {
	//			log.Fatal("Fatal error creating def.js: ", err)
	//		//}
	//	}
	//	if _, err := f.WriteString(jsTxt); err != nil {
	//		_ = f.Close()
	//		log.Fatal("Fatal error writing to op.js: ", err)
	//	}
	//	if _, err := f.WriteString(jsST); err != nil {
	//		_ = f.Close()
	//		log.Fatal("Fatal error writing to op.js: ", err)
	//	}
	//	if _, err := f.WriteString(jsEN); err != nil {
	//		_ = f.Close()
	//		log.Fatal("Fatal error writing to op.js: ", err)
	//	}
	//	if err = f.Close(); err != nil {
	//		log.Fatal("Fatal error closing op.js: ", err)
	//	}
	//}

}

// set-up DBS thread and data flow structure based on the provided configuration
// it needs to be modified in order to supp all foreseen data
func setUpDataDBSBank(spaceChannels map[string]chan spaceEntries) {

	now := supp.Timestamp()

	latestChannelLock.Lock()
	LatestBankOut = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	latestBankIn = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	latestDBSIn = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	//_ResetDBS = make(map[string]map[string]map[string]chan bool, len(spaceChannels))

	for dl, dt := range dataTypes {

		// Add all possible processing data thread needing the database
		switch dl {
		case "presence":
			// set-ups the database for the presence threads is done in the thread self,
			// since it provide presence only after the period and not in real time
		default:
			LatestBankOut[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))
			latestBankIn[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))

			//if supp.Debug < 3 {
			latestDBSIn[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))
			//_ResetDBS[dl] = make(map[string]map[string]chan bool, len(spaceChannels)) // placeholder
			//}
			// set-ups the database for the sampler and average processing DBS thread
			for name := range spaceChannels {
				LatestBankOut[dl][name] = make(map[string]chan interface{}, len(AvgAnalysis))
				latestBankIn[dl][name] = make(map[string]chan interface{}, len(AvgAnalysis))
				//if supp.Debug < 3 {
				//_ResetDBS[dl][name] = make(map[string]chan bool, len(avgAnalysis)) // placeholder
				latestDBSIn[dl][name] = make(map[string]chan interface{}, len(AvgAnalysis))
				//}
				for _, v := range AvgAnalysis {
					LatestBankOut[dl][name][v.Name] = make(chan interface{})
					latestBankIn[dl][name][v.Name] = make(chan interface{})
					// formatting of initialisation data
					if len(InitData[dl][name][v.Name]) == 2 {
						// possibly valid InitData data found
						tag := dl + name + v.Name
						if ts, err := strconv.ParseInt(InitData[dl][name][v.Name][0], 10, 64); err == nil {
							// found valid InitData data
							if (now - ts) < CrashMaxDelay {
								// data is fresh enough
								switch dl {
								case "sample__":
									if va, e := strconv.Atoi(InitData[dl][name][v.Name][1]); e == nil {
										go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.Name], LatestBankOut[dl][name][v.Name], DataEntry{id: tag, Ts: ts, NetFlow: va})
									} else {
										log.Printf("spaces.setUpDataDBSBank: invalid InitData data for %v\n", tag)
										go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.Name], LatestBankOut[dl][name][v.Name])
									}
								case "entry___":
									vas := strings.Split(InitData[dl][name][v.Name][1][2:len(InitData[dl][name][v.Name][1])-2], "][")
									var va [][]int
									for _, el := range vas {
										sd := strings.Split(el, " ")
										if len(sd) == 4 {
											sd0, e0 := strconv.Atoi(sd[0])
											sd1, e1 := strconv.Atoi(sd[1])
											sd2, e2 := strconv.Atoi(sd[2])
											sd3, e3 := strconv.Atoi(sd[3])
											if e0 == nil && e1 == nil && e2 == nil && e3 == nil {
												va = append(va, []int{sd0, sd1, sd2, sd3})
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
										go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.Name], LatestBankOut[dl][name][v.Name], data)
									} else {
										log.Printf("spaces.setUpDataDBSBank: invalid InitData data for %v\n", tag)
										go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.Name], LatestBankOut[dl][name][v.Name])
									}
								default:
									log.Printf("spaces.setUpDataDBSBank: invalid InitData data type for %v\n", tag)
									go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.Name], LatestBankOut[dl][name][v.Name])
								}
							} else {
								// data is not fresh enough
								log.Printf("spaces.setUpDataDBSBank: too old InitData Ts for %v\n", tag)
								go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.Name], LatestBankOut[dl][name][v.Name])
							}
						} else {
							log.Printf("spaces.setUpDataDBSBank: invalid InitData Ts for %v\n", tag)
							go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.Name], LatestBankOut[dl][name][v.Name])
						}
					} else {
						// register started with no InitData data (not available)
						go storage.SafeRegGeneric(dl+name+v.Name, latestBankIn[dl][name][v.Name], LatestBankOut[dl][name][v.Name])
					}

					//if supp.Debug < 3 {
					// Start of distributed data passing structure
					// the reset channel is not used at the moment, this is a place holder
					//_ResetDBS[dl][name][v.name] = make(chan bool)
					latestDBSIn[dl][name][v.Name] = make(chan interface{})
					label := dl + name + v.Name
					if _, e := storage.SetSeries(label, v.Interval, !supp.StringEnding(label, "current", "_")); e != nil {
						log.Fatalf("spaces.setUpDataDBSBank: fatal error setting database %v:%v\n", name+v.Name, v.Interval)
					}
					//go dt.cf(dl+name+v.name, latestDBSIn[dl][name][v.name], _ResetDBS[dl][name][v.name])
					go dt.cf(dl+name+v.Name, latestDBSIn[dl][name][v.Name], nil)
					//}
				}
				log.Printf("spaces.setUpDataDBSBank: DataBank for space %v and data %v initialised\n", name, dl)
			}
		}
	}
	latestChannelLock.Unlock()
}
