package spaces

import (
	"log"
	"os"
	"strconv"
	"strings"
)

func SetUp() {
	spaceChannels = make(map[string]chan dataChan)
	gateChannels = make(map[int][]chan dataChan)
	GroupsStats = make(map[int]int)
	gateGroup = make(map[int]int)

	if data := os.Getenv("REVERSE"); data != "" {
		for _, v := range strings.Split(data, " ") {
			if vi, e := strconv.Atoi(v); e == nil {
				reversedGates = append(reversedGates, vi)
			} else {
				log.Fatal("spaces.SetUp: fatal error converting reversed gate name", v)
			}
		}
		log.Println("spaces.SetUp: defined reversed gates", reversedGates)
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
						log.Fatal("spaces.SetUp: fatal error gate name", val)
					}
				}
				log.Printf("spaces.SetUp: found space [%v] with gates %v\n", name, sg)
				for _, g := range sg {
					gateChannels[g] = append(gateChannels[g], spaceChannels[name])
				}
			} else {
				log.Printf("spaces.SetUp: found  empty space [%v]\n", name)
			}
		}
	} else {
		log.Fatal("spaces.SetUp: fatal error no space has been defined")
	}

	if data := os.Getenv("GROUPS"); data != "" {
		for _, gg := range strings.Split(data, ";") {
			gg = strings.Trim(gg, " ")
			tempg := strings.Split(gg, " ")
			tgn, e := strconv.Atoi(tempg[0])
			if e != nil {
				log.Fatal("spaces.SetUp: fatal error group name", tempg[0])
			} else {
				log.Printf("spaces.SetUp: found group with ID:%v\n", tempg[0])
			}
			tempg = tempg[1:]
			for _, v := range tempg {
				if val, e := strconv.Atoi(strings.Trim(v, " ")); e != nil {
					log.Fatal("spaces.SetUp: fatal error group gate name", v)
				} else {
					if _, pres := gateGroup[val]; pres {
						log.Println("spaces.SetUp: error group duplicated gate name", val)
					} else {
						gateGroup[val] = tgn
					}
				}
			}
		}

		// Initialise the statistics on groups
		for _, v := range gateGroup {
			if _, ok := GroupsStats[v]; ok {
				GroupsStats[v] += 1
			} else {
				GroupsStats[v] = 1
			}
		}
	}

}

// TODO sets up the counters - in progress
func CountersSetpUp() {
	sw := os.Getenv("SAMWINDOW")
	if sw == "" {
		samplingWindow = 30
	} else {
		if v, e := strconv.Atoi(sw); e != nil {
			log.Fatal("spaces.CountersSetpUp: fatal error in definition of SAMWINDOW")
		} else {
			samplingWindow = int64(v)
		}
	}
	log.Printf("spaces.CountersSetpUp: setting sliding window at %vs\n", samplingWindow)
}
