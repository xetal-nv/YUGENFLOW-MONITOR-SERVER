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

	// all possible data to be stored and distributed needs to have an entry in dtypes
	// the data type needs to implement the following interfaces:
	//  storage.sampledata, servers.genericdata
	// currently defined are: sample, entry, presence
	// track usage of dtypes and use proper conditional statements

	// add sample type based on DataEntry
	var sample = dtfuncs{}
	sample.pf = func(nm string, se spaceEntries) interface{} {
		return DataEntry{id: nm, Ts: se.ts, Val: se.val}
	}
	sample.cf = func(id string, in chan interface{}, rst chan bool) {
		storage.SerieSampleDBS(id, in, rst, "TS")
	}
	dtypes[support.StringLimit("sample", support.LabelLength)] = sample
	// add entry type
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
				entries = append(entries, []int{id, v.Val})
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
	dtypes[support.StringLimit("entry", support.LabelLength)] = entry
	// add presence type, which is equal to sample
	var presence = dtfuncs{}
	presence.pf = func(nm string, se spaceEntries) interface{} {
		return DataEntry{id: nm, Ts: se.ts, Val: se.val}
	}
	presence.cf = func(id string, in chan interface{}, rst chan bool) {
		storage.SerieSampleDBS(id, in, rst, "SD")
	}
	dtypes[support.StringLimit("presence", support.LabelLength)] = presence

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
			// filter out data that do not need Start value form recovery
			switch i {
			case "presence":
				// skip
			default:
				// define
				InitData[i] = make(map[string]map[string][]string)
				for _, j := range spaces {
					name := strings.Trim(strings.Split(j, ":")[0], " ")
					InitData[i][support.StringLimit(name, support.LabelLength)] = make(map[string][]string)
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
				data := strings.Split(scanner.Text(), ",")
				if len(data) == 5 {
					// any other length implies a wrong data and will be ignored
					InitData[data[0]][data[1]][data[2]] = []string{data[3], data[4]}
				}
			}
			log.Printf("spaces.setUpDataDBSBank: InitData file imported\n")
		}
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
	spacePresenceChannels := make(map[string]chan spaceEntries)
	entrySpaceSamplerChannels = make(map[int][]chan spaceEntries)
	entrySpacePresenceChannels = make(map[int][]chan spaceEntries)
	SpaceDef = make(map[string][]int)
	SpaceTimes = make(map[string]TimeSchedule)

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

		// initialise the processing threads for each space
		for _, name := range spaces {
			var sprange TimeSchedule
			nl, _ := time.Parse(support.TimeLayout, "00:00")
			rng := strings.Split(strings.Trim(os.Getenv("CLOSURE_"+name), ";"), ";")
			sprange.Start, sprange.End = nl, nl
			if len(rng) == 2 {
				for i, v := range rng {
					rng[i] = strings.Trim(v, " ")
				}
				if v, e := time.Parse(support.TimeLayout, rng[0]); e == nil {
					sprange.Start = v
					if v, e := time.Parse(support.TimeLayout, rng[1]); e == nil {
						sprange.End = v
					} else {
						sprange.Start = nl
					}
				}
			}
			SpaceTimes[support.StringLimit(name, support.LabelLength)] = sprange
			LatestDetectorOut = make(map[string]chan []IntervalDetector)
			if sts := os.Getenv("SPACE_" + name); sts != "" {
				name = support.StringLimit(name, support.LabelLength)

				// the go routines below Start the relevant processing threads
				// sampling and averaging threads
				spaceChannels[name] = make(chan spaceEntries, bufsize)
				go sampler(name, spaceChannels[name], nil, nil, nil, 0, sync.Once{}, 0, 0)

				// presence detection threads
				spacePresenceChannels[name] = make(chan spaceEntries, bufsize)
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

	return spaceChannels
}

// set-up counters thread and data flow structure based on the provided configuration
func setpUpCounter() {
	cmode = os.Getenv("CMODE")
	if cmode == "" {
		cmode = "0"
	}
	log.Printf("Compression mode set to %v\n", cmode)

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

	avgw := strings.Trim(os.Getenv("ANALYSISPERIOD"), ";")
	avgWindows := make(map[string]int)
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
	} else {
		log.Fatal("spaces.setpUpCounter: no averaging values have been defined in ANALYSISPERIOD!!!")
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

	if val := strings.Split(strings.Trim(os.Getenv("ANALYSISWINDOW"), " "), " "); len(val) == 2 {
		if st, e := time.Parse(support.TimeLayout, val[0]); e == nil {
			if en, e := time.Parse(support.TimeLayout, val[1]); e == nil {
				avgAnalysisSchedule = TimeSchedule{st, en, 0}
				avgAnalysisSchedule.Duration, _ = support.TimeDifferenceInSecs(val[0], val[1])
				avgAnalysisSchedule.Duration += 60000
				log.Printf("spaces.setpUpCounter: Analysis window is set from %v to %v\n", val[0], val[1])
			} else {
				log.Fatal("spaces.setpUpCounter: illegal End ANALYSISWINDOW value", val)
			}
		} else {
			log.Fatal("spaces.setpUpCounter: illegal Start ANALYSISWINDOW value", val)
		}
	}

}

// set-up DBS thread and data flow structure based on the provided configuration
// it needs to be modified in order to support all foreseen data
func setUpDataDBSBank(spaceChannels map[string]chan spaceEntries) {

	now := support.Timestamp()

	latestChannelLock.Lock()
	LatestBankOut = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	latestBankIn = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	latestDBSIn = make(map[string]map[string]map[string]chan interface{}, len(spaceChannels))
	//_ResetDBS = make(map[string]map[string]map[string]chan bool, len(spaceChannels))

	for dl, dt := range dtypes {

		// Add all possible processing data thread needing the database
		switch dl {
		case "presence":
			// set-ups the database for the presence threads is done in the thread self,
			// since it provide presence only after the period and not in real time
		default:
			LatestBankOut[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))
			latestBankIn[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))

			if support.Debug < 3 {
				latestDBSIn[dl] = make(map[string]map[string]chan interface{}, len(spaceChannels))
				//_ResetDBS[dl] = make(map[string]map[string]chan bool, len(spaceChannels)) // placeholder
			}
			// set-ups the database for the sampler and average processing DBS thread
			for name := range spaceChannels {
				LatestBankOut[dl][name] = make(map[string]chan interface{}, len(avgAnalysis))
				latestBankIn[dl][name] = make(map[string]chan interface{}, len(avgAnalysis))
				if support.Debug < 3 {
					//_ResetDBS[dl][name] = make(map[string]chan bool, len(avgAnalysis)) // placeholder
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
								switch dl {
								case "sample__":
									if va, e := strconv.Atoi(InitData[dl][name][v.name][1]); e == nil {
										go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name], DataEntry{tag, ts, va})
									} else {
										log.Printf("spaces.setUpDataDBSBank: invalid InitData data for %v\n", tag)
										go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
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
										go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name], data)
									} else {
										log.Printf("spaces.setUpDataDBSBank: invalid InitData data for %v\n", tag)
										go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
									}
								default:
									log.Printf("spaces.setUpDataDBSBank: invalid InitData data type for %v\n", tag)
									go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
								}
							} else {
								// data is not fresh enough
								log.Printf("spaces.setUpDataDBSBank: too old InitData Ts for %v\n", tag)
								go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
							}
						} else {
							log.Printf("spaces.setUpDataDBSBank: invalid InitData Ts for %v\n", tag)
							go storage.SafeRegGeneric(tag, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
						}
					} else {
						// register started with no InitData data (not available)
						go storage.SafeRegGeneric(dl+name+v.name, latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
					}

					if support.Debug < 3 {
						// Start of distributed data passing structure
						// the reset channel is not used at the moment, this is a place holder
						//_ResetDBS[dl][name][v.name] = make(chan bool)
						latestDBSIn[dl][name][v.name] = make(chan interface{})
						label := dl + name + v.name
						if _, e := storage.SetSeries(label, v.interval, !support.Stringending(label, "current", "_")); e != nil {
							log.Fatal("spaces.setUpDataDBSBank: fatal error setting database " + name + v.name + ", " + strconv.Itoa(v.interval))
						}
						//go dt.cf(dl+name+v.name, latestDBSIn[dl][name][v.name], _ResetDBS[dl][name][v.name])
						go dt.cf(dl+name+v.name, latestDBSIn[dl][name][v.name], nil)
					}
				}
				log.Printf("spaces.setUpDataDBSBank: DataBank for space %v and data %v initialised\n", name, dl)
			}
		}
	}
	latestChannelLock.Unlock()
}
