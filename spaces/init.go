package spaces

import (
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type schan struct {
	num   int
	val   int
	group int
}

var sp map[string]chan schan // space list
var ag map[int][]chan schan  // sensor list
var g2g map[int]int          // maps gate to group_id

var GroupsStats map[int]int // gives size og group per group_id

func SetUp() {
	sp = make(map[string]chan schan)
	ag = make(map[int][]chan schan)
	GroupsStats = make(map[int]int)
	g2g = make(map[int]int)
	spaces := strings.Split(os.Getenv("SPACES"), " ")
	for i := 0; i < len(spaces); i++ {
		spaces[i] = strings.Trim(spaces[i], " ")
	}
	for _, gg := range strings.Split(os.Getenv("GROUPS"), ";") {
		gg = strings.Trim(gg, " ")
		tempg := strings.Split(gg, " ")
		tgn, e := strconv.Atoi(tempg[0])
		if e != nil {
			log.Fatal("Spaces SetUp: fatal error group name ", tempg[0])
		}
		tempg = tempg[1:]
		for _, v := range tempg {
			if val, e := strconv.Atoi(strings.Trim(v, " ")); e != nil {
				log.Fatal("Spaces SetUp: fatal error group gate name ", v)
			} else {
				if _, pres := g2g[val]; pres {
					log.Println("Spaces SetUp: error group duplicated gate name ", val)
				} else {
					g2g[val] = tgn
				}
			}
		}
	}
	for _, v := range g2g {
		if _, ok := GroupsStats[v]; ok {
			GroupsStats[v] += 1
		} else {
			GroupsStats[v] = 1
		}
	}
	for i, name := range spaces {
		sp[name] = make(chan schan)
		go saveToFile(name)
		var sg []int
		for _, val := range strings.Split(os.Getenv("GATES_"+strconv.Itoa(i)), " ") {
			vt := strings.Trim(val, " ")
			if v, e := strconv.Atoi(vt); e == nil {
				sg = append(sg, v)
			} else {
				log.Fatal("Spaces SetUp: fatal error gate name ", val)
			}
		}
		log.Printf("setUpSpaces: found space [%v] with gates %v\n", name, sg)
		for _, g := range sg {
			ag[g] = append(ag[g], sp[name])
		}
	}

	// DEBUG
	//fmt.Println("SPACES:", sp)
	//fmt.Println("GATES", ag)
	//fmt.Println("GROUPS STATS:", GroupsStats)
	//fmt.Println("GATE2GROUP", g2g)
	//os.Exit(0)
}

// TODO
// sets up the counters
//func CountersSetpUp() {
//	time.Sleep(1 * time.Second)
//}

// saves raw data to a file
func saveToFile(spn string) {
	c := sp[spn]
	if c == nil {
		log.Printf("Spaces.saveToFile: error space %v not valid\n", spn)
	} else {
		log.Printf("Spaces.saveToFile: enabled space [%v]\n", spn)
		var resultf *os.File
		var e error
		if resultf, e = os.OpenFile(spn+".rawdata", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); e != nil {
			log.Fatal(e)
		}
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Printf("saveToFile: recovering for gate %+v from: %v\n ", c, e)
					//noinspection GoUnhandledErrorResult
					resultf.Close()
					go saveToFile(spn)
				}
			}
		}()
		for {
			val := <-c
			if _, e := fmt.Fprintf(resultf, "%v, %v, %v, %v\n", time.Now().UTC().Unix(), val.num, val.val,
				val.group); e != nil {
				log.Printf("Spaces.saveToFile: error space %v not valid\n", spn)
			}
		}
	}
}

// sends the gate data to the proper counters
func SendData(gate int, val int) error {
	if ag[gate] == nil {
		return errors.New("Spaces.SendData: error gate not valid")
	}
	for _, v := range ag[gate] {
		go func() { v <- schan{num: gate, val: val, group: g2g[gate]} }()
	}
	return nil
}
