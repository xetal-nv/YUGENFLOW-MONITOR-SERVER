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
	setpUpCounter()
	spchans := setUpSpaces()
	setUpDataDBSBank(spchans)
}

func setUpSpaces() map[string]chan interface{} {
	spaceChannels := make(map[string]chan interface{})
	entrySpaceChannels = make(map[int][]chan interface{})
	//groupsStats = make(map[int]int)
	//entryGroup = make(map[int]int)

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
				spaceChannels[name] = make(chan interface{}, bufsize)
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

func setUpDataDBSBank(spaceChannels map[string]chan interface{}) {

	//LatestDataBankOut = make(map[string]map[string]chan storage.DataCt, len(spaceChannels))
	LatestDataBankOut = make(map[string]map[string]chan interface{}, len(spaceChannels))
	//latestDataBankIn = make(map[string]map[string]chan storage.DataCt, len(spaceChannels))
	latestDataBankIn = make(map[string]map[string]chan interface{}, len(spaceChannels))
	if support.Debug < 3 {
		//latestDataDBSIn = make(map[string]map[string]chan storage.DataCt, len(spaceChannels))
		latestDataDBSIn = make(map[string]map[string]chan interface{}, len(spaceChannels))
		ResetDataDBS = make(map[string]map[string]chan bool, len(spaceChannels))
	}

	for name := range spaceChannels {
		//LatestDataBankOut[name] = make(map[string]chan storage.DataCt, len(avgAnalysis))
		LatestDataBankOut[name] = make(map[string]chan interface{}, len(avgAnalysis))
		//latestDataBankIn[name] = make(map[string]chan storage.DataCt, len(avgAnalysis))
		latestDataBankIn[name] = make(map[string]chan interface{}, len(avgAnalysis))
		if support.Debug < 3 {
			ResetDataDBS[name] = make(map[string]chan bool, len(avgAnalysis))
			//latestDataDBSIn[name] = make(map[string]chan storage.DataCt, len(avgAnalysis))
			latestDataDBSIn[name] = make(map[string]chan interface{}, len(avgAnalysis))
		}
		for _, v := range avgAnalysis {
			//LatestDataBankOut[name][v.name] = make(chan storage.DataCt)
			LatestDataBankOut[name][v.name] = make(chan interface{})
			//latestDataBankIn[name][v.name] = make(chan storage.DataCt)
			latestDataBankIn[name][v.name] = make(chan interface{})
			//go storage.TimedIntCell(name+v.name, latestDataBankIn[name][v.name], LatestDataBankOut[name][v.name])
			go storage.SafeReg(latestDataBankIn[name][v.name], LatestDataBankOut[name][v.name])
			if support.Debug < 3 {
				ResetDataDBS[name][v.name] = make(chan bool)
				//latestDataDBSIn[name][v.name] = make(chan storage.DataCt)
				latestDataDBSIn[name][v.name] = make(chan interface{})
				if _, e := storage.SetSeries(name+v.name, v.interval, false); e != nil {
					log.Fatal("spaces.setUpDataDBSBank: fatal error setting database %v:%v\n", name+v.name, v.interval)
					//go storage.TimedIntDBS(name+v.name, latestDataBankIn[name][v.name], ResetDataDBS[name][v.name])
					go storage.SerieSampleDBS(name+v.name, latestDataDBSIn[name][v.name], ResetDataDBS[name][v.name])
				}
			}
		}
		log.Printf("spaces.setUpDataDBSBank: DataBank for space %v initialised\n", name)
	}

}
