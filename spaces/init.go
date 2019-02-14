package spaces

import (
	"countingserver/registers"
	"log"
	"os"
	"strconv"
	"strings"
)

func SetUp() {
	setUpSpaces()
	setpUpCounter()
	setUpDataBank()
}

func setUpSpaces() {
	spaceChannels = make(map[string]chan dataChan)
	gateChannels = make(map[int][]chan dataChan)
	groupsStats = make(map[int]int)
	gateGroup = make(map[int]int)

	if data := os.Getenv("REVERSE"); data != "" {
		for _, v := range strings.Split(data, " ") {
			if vi, e := strconv.Atoi(v); e == nil {
				reversedGates = append(reversedGates, vi)
			} else {
				log.Fatal("spaces.setUpSpaces: fatal error converting reversed gate name", v)
			}
		}
		log.Println("spaces.setUpSpaces: defined reversed gates", reversedGates)
	}

	if data := os.Getenv("SPACES"); data != "" {
		spaces := strings.Split(data, " ")
		bufsize := 50
		if v, e := strconv.Atoi(os.Getenv("INTBUFFSIZE")); e == nil {
			bufsize = v
		}

		for i := 0; i < len(spaces); i++ {
			spaces[i] = strings.Trim(spaces[i], " ")
		}

		// initialise the processing threads for each space
		for i, name := range spaces {
			if gts := os.Getenv("GATES_" + strconv.Itoa(i)); gts != "" {
				spaceChannels[name] = make(chan dataChan, bufsize)
				// the go routine below is the processing thread.
				go sampler(name)
				var sg []int
				for _, val := range strings.Split(gts, " ") {
					vt := strings.Trim(val, " ")
					if v, e := strconv.Atoi(vt); e == nil {
						sg = append(sg, v)
					} else {
						log.Fatal("spaces.setUpSpaces: fatal error gate name", val)
					}
				}
				log.Printf("spaces.setUpSpaces: found space [%v] with gates %v\n", name, sg)
				for _, g := range sg {
					gateChannels[g] = append(gateChannels[g], spaceChannels[name])
				}
			} else {
				log.Printf("spaces.setUpSpaces: found  empty space [%v]\n", name)
			}
		}
	} else {
		log.Fatal("spaces.setUpSpaces: fatal error no space has been defined")
	}

	if data := os.Getenv("GROUPS"); data != "" {
		for _, gg := range strings.Split(data, ";") {
			gg = strings.Trim(gg, " ")
			tempg := strings.Split(gg, " ")
			tgn, e := strconv.Atoi(tempg[0])
			if e != nil {
				log.Fatal("spaces.setUpSpaces: fatal error group name", tempg[0])
			} else {
				log.Printf("spaces.setUpSpaces: found group with ID:%v\n", tempg[0])
			}
			tempg = tempg[1:]
			for _, v := range tempg {
				if val, e := strconv.Atoi(strings.Trim(v, " ")); e != nil {
					log.Fatal("spaces.setUpSpaces: fatal error group gate name", v)
				} else {
					if _, pres := gateGroup[val]; pres {
						log.Println("spaces.setUpSpaces: error group duplicated gate name", val)
					} else {
						gateGroup[val] = tgn
					}
				}
			}
		}

		// Initialise the statistics on groups
		for _, v := range gateGroup {
			if _, ok := groupsStats[v]; ok {
				groupsStats[v] += 1
			} else {
				groupsStats[v] = 1
			}
		}
	}

}

func setpUpCounter() {
	sw := os.Getenv("SAMWINDOW")
	if os.Getenv("INSTANTNEG") == "1" {
		negSkip = false
	} else {
		negSkip = true
	}
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

	avgw := os.Getenv("SAVEWINDOW")
	avgWindows = make(map[string]int)
	avgWindows["current"] = samplingWindow
	if avgw != "" {
		for _, v := range strings.Split(avgw, ";") {
			data := strings.Split(strings.Trim(v, " "), " ")
			if (len(data)) != 2 {
				log.Fatal("spaces.setpUpCounter: fatal error for illegal SAVEWINDOW values", data)
			}
			if v, e := strconv.Atoi(data[1]); e != nil {
				log.Fatal("spaces.setpUpCounter: fatal error for illegal SAVEWINDOW values", data)
			} else {
				avgWindows[data[0]] = v
			}
		}
	}
	log.Printf("spaces.setpUpCounter: setting averaging windows at %v\n", avgWindows)
}

func setUpDataBank() {
	LatestDataBankOut = make(map[string]map[string]chan registers.DataCt, len(spaceChannels))
	LatestDataBankIn = make(map[string]map[string]chan registers.DataCt, len(spaceChannels))

	for name, _ := range spaceChannels {
		LatestDataBankOut[name] = make(map[string]chan registers.DataCt, len(avgWindows))
		LatestDataBankIn[name] = make(map[string]chan registers.DataCt, len(avgWindows))
		for id, _ := range avgWindows {
			LatestDataBankOut[name][id] = make(chan registers.DataCt)
			LatestDataBankIn[name][id] = make(chan registers.DataCt)
			go registers.TimedIntCell(name+id, LatestDataBankIn[name][id], LatestDataBankOut[name][id])
		}
		log.Println("spaces.setUpDataBank: DataBank initialised")
	}

}
