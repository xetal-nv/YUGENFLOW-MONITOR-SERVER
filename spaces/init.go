package spaces

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO add gate orientation information
func SetUp() {
	spaceChannels = make(map[string]chan dataChan)
	gateChannels = make(map[int][]chan dataChan)
	GroupsStats = make(map[int]int)
	gateGroup = make(map[int]int)
	spaces := strings.Split(os.Getenv("SPACES"), " ")
	for _, v := range strings.Split(os.Getenv("REVERSE"), " ") {
		if vi, e := strconv.Atoi(v); e == nil {
			reversedGates = append(reversedGates, vi)
		} else {
			log.Fatal("spaces.SetUp: fatal error converting reversed gate name", v)
		}
	}
	log.Println("spaces.SetUp: defined reversed gates", reversedGates)
	for i := 0; i < len(spaces); i++ {
		spaces[i] = strings.Trim(spaces[i], " ")
	}
	for _, gg := range strings.Split(os.Getenv("GROUPS"), ";") {
		gg = strings.Trim(gg, " ")
		tempg := strings.Split(gg, " ")
		tgn, e := strconv.Atoi(tempg[0])
		if e != nil {
			log.Fatal("spaces.SetUp: fatal error group name", tempg[0])
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

	// initialise the processing threads for each space
	for i, name := range spaces {
		spaceChannels[name] = make(chan dataChan)

		// the go routine below is the processing thread.
		go Counters(name)

		var sg []int
		for _, val := range strings.Split(os.Getenv("GATES_"+strconv.Itoa(i)), " ") {
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
	}

	// DEBUG
	//fmt.Println("SPACES:", spaceChannels)
	//fmt.Println("GATES", gateChannels)
	//fmt.Println("GROUPS STATS:", GroupsStats)
	//fmt.Println("GATE2GROUP", gateGroup)
	//os.Exit(0)
}

// TODO sets up the counters
func CountersSetpUp() {
	time.Sleep(1 * time.Second)
}
