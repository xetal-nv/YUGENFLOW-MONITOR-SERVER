package spaces

import (
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
	spaceTimes = make(map[string]closureRange)

	if data := os.Getenv("SPACES_NAMES"); data != "" {
		spaces := strings.Split(data, " ")
		bufsize = 50
		if v, e := strconv.Atoi(os.Getenv("INTBUFFSIZE")); e == nil {
			bufsize = v
		}

		for i := 0; i < len(spaces); i++ {
			spaces[i] = strings.Trim(spaces[i], " ")
		}

		// initialise the processing threads for each space
		for _, name := range spaces {
			var sprange closureRange
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
			//if sts := os.Getenv("SPACE_" + strconv.Itoa(i)); sts != "" {
			if sts := os.Getenv("SPACE_" + name); sts != "" {
				name = support.StringLimit(name, support.LabelLength)
				spaceChannels[name] = make(chan spaceEntries, bufsize)
				// the go routine below is the processing thread.
				go sampler(name, spaceChannels[name], nil, 0, sync.Once{}, 0, 0)
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
	return spaceChannels
}

// set-up counters thread and data flow structure based on the provided configuration
func setpUpCounter() {
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

	avgw := strings.Trim(os.Getenv("SAVEWINDOW"), ";")
	avgWindows := make(map[string]int)
	tw := make(map[int]string)
	curr := support.StringLimit("current", support.LabelLength)
	avgWindows[curr] = SamplingWindow
	tw[SamplingWindow] = curr
	if avgw != "" {
		for _, v := range strings.Split(avgw, ";") {
			data := strings.Split(strings.Trim(v, " "), " ")
			if (len(data)) != 2 {
				log.Fatal("spaces.setpUpCounter: fatal error for illegal SAVEWINDOW values", data)
			}
			if v, e := strconv.Atoi(data[1]); e != nil {
				log.Fatal("spaces.setpUpCounter: fatal error for illegal SAVEWINDOW values", data)
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
}

// set-up DBS thread and data flow structure based on the provided configuration
func setUpDataDBSBank(spaceChannels map[string]chan spaceEntries) {

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
				go storage.SafeReg(latestBankIn[dl][name][v.name], LatestBankOut[dl][name][v.name])
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
}
