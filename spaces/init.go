package spaces

import (
	"countingserver/storage"
	"countingserver/support"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

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
	dtypes["sample"] = sample
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
	dtypes["entry"] = entry
	// set up the server for data analysis/transmission/storage
	setpUpCounter()
	spchans := setUpSpaces()
	setUpDataDBSBank(spchans)
}

func setUpSpaces() (spaceChannels map[string]chan spaceEntries) {
	spaceChannels = make(map[string]chan spaceEntries)
	entrySpaceChannels = make(map[int][]chan spaceEntries)

	if data := os.Getenv("SPACES_NAMES"); data != "" {
		spaces := strings.Split(data, " ")
		bufsize = 50
		if v, e := strconv.Atoi(os.Getenv("INTBUFFSIZE")); e == nil {
			bufsize = v
		}

		for i := 0; i < len(spaces); i++ {
			spaces[i] = strings.Trim(spaces[i], " ")
		}

		//onces := make([]sync.Once, len(spaces))

		// initialise the processing threads for each space
		for _, name := range spaces {
			//if sts := os.Getenv("SPACE_" + strconv.Itoa(i)); sts != "" {
			if sts := os.Getenv("SPACE_" + name); sts != "" {
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

func setpUpCounter() {
	sw := os.Getenv("SAMWINDOW")
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

	if sw == "" {
		samplingWindow = 30
	} else {
		if v, e := strconv.Atoi(sw); e != nil {
			log.Fatal("spaces.setpUpCounter: fatal error in definition of SAMWINDOW")
		} else {
			samplingWindow = v
			//}
		}
	}
	log.Printf("spaces.setpUpCounter: setting sliding window at %vs\n", samplingWindow)

	avgw := strings.Trim(os.Getenv("SAVEWINDOW"), ";")
	avgWindows := make(map[string]int)
	tw := make(map[int]string)
	avgWindows["current"] = samplingWindow
	tw[samplingWindow] = "current"
	if avgw != "" {
		for _, v := range strings.Split(avgw, ";") {
			data := strings.Split(strings.Trim(v, " "), " ")
			if (len(data)) != 2 {
				log.Fatal("spaces.setpUpCounter: fatal error for illegal SAVEWINDOW values", data)
			}
			if v, e := strconv.Atoi(data[1]); e != nil {
				log.Fatal("spaces.setpUpCounter: fatal error for illegal SAVEWINDOW values", data)
			} else {
				if v > samplingWindow {
					avgWindows[data[0]] = v
					tw[v] = data[0]
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
					if _, e := storage.SetSeries(dl+name+v.name, v.interval, false); e != nil {
						log.Fatal("spaces.setUpDataDBSBank: fatal error setting database %v:%v\n", name+v.name, v.interval)
					}
					go dt.cf(dl+name+v.name, latestDBSIn[dl][name][v.name], ResetDBS[dl][name][v.name])
				}
			}
			log.Printf("spaces.setUpDataDBSBank: DataBank for space %v and data %v initialised\n", name, dl)
		}
	}
}
