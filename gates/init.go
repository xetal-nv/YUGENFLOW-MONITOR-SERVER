package gates

import (
	"countingserver/support"
	"log"
	"os"
	"strconv"
	"strings"
)

func SetUp() {
	var revdev []int
	devgate := [2]int{2, 2}
	if data := os.Getenv("DEVPERGATE"); data != "" {
		v := strings.Split(strings.Trim(data, " "), " ")
		if len(v) != 2 {
			log.Fatal("gateList.SetUp: fatal error, illegal number of parameters in DEVPERGATE")
		} else {
			if ind, ok := strconv.Atoi(v[0]); ok != nil {
				log.Fatal("gateList.SetUp: fatal error,  illegal parameter in DEVPERGATE", v[0])
			} else {
				devgate[0] = ind
			}
			if ind, ok := strconv.Atoi(v[1]); ok != nil {
				log.Fatal("gateList.SetUp: fatal error,  illegal parameter in DEVPERGATE ", v[1])
			} else {
				devgate[1] = ind
			}
		}
	}
	log.Println("gateList.SetUp: defined gate composition range in devices as", devgate)
	if data := os.Getenv("REVERSE"); data != "" {
		for _, v := range strings.Split(data, " ") {
			if vi, e := strconv.Atoi(v); e == nil {
				revdev = append(revdev, vi)
			} else {
				log.Fatal("gateList.SetUp: fatal error converting reversed gate name ", v)
			}
		}
		log.Println("gateList.SetUp: defined reversed gates", revdev)
	}
	i := 0
	if data := os.Getenv("GATE_" + strconv.Itoa(i)); data == "" {
		log.Fatal("gateList.SetUp: fatal error, no gate has been defined")
	} else {
		sensorList = make(map[int]sensorDef)
		gateList = make(map[int][]int)
		for data != "" {
			t := strings.Split(strings.Trim(data, " "), " ")
			if len(t) < devgate[0] || len(t) > devgate[1] {
				log.Fatal("gateList.SetUp: fatal error, illegal number of deviced for gate ", i)
			}
			for _, v := range t {
				if ind, ok := strconv.Atoi(v); ok != nil {
					log.Fatal("gateList.SetUp: fatal error in definition of GATE ", i)
				} else {
					rev := false
					if support.Contains(revdev, ind) {
						rev = true
					}
					if v, ok := sensorList[ind]; ok {
						v.gate = append(v.gate, i)
						sensorList[ind] = v
					} else {
						sensorList[ind] = sensorDef{id: ind, reversed: rev, gate: []int{i}}
					}

					gateList[i] = append(gateList[i], ind)
				}
			}
			log.Printf("gateList.SetUp: defined gate %v as [id reversed]:\n", i)
			for _, v := range gateList[i] {
				log.Printf("\t\t [%v %v]\n", sensorList[v].id, sensorList[v].reversed)
			}
			i += 1
			data = os.Getenv("GATE_" + strconv.Itoa(i))
		}
	}
	i = 0
	if data := os.Getenv("ENTRY_" + strconv.Itoa(i)); data == "" {
		log.Fatal("gateList.SetUp: fatal error, no entry has been defined")
	} else {
		entryList = make(map[int]entryDef)
		for data != "" {
			t := entryDef{id: 1}
			t.senDef = make(map[int]sensorDef)
			t.gates = make(map[int][]int)
			entryChan := make(chan sensorData)
			for _, v := range strings.Split(strings.Trim(data, " "), " ") {
				if ind, ok := strconv.Atoi(v); ok != nil {
					log.Fatal("gateList.SetUp: fatal error in definition of ENTRY ", i)
				} else {
					t.gates[ind] = gateList[ind]
					for _, d := range gateList[ind] {
						//entryList[i] = append(entryList[i], d)
						tm := sensorList[d]
						kp := true
						for _, v := range tm.entry {
							if v == entryChan {
								kp = false
							}
						}
						if kp {
							tm.entry = append(tm.entry, entryChan)
							sensorList[d] = tm
						}
						t.senDef[d] = sensorList[d]
					}
				}
			}
			entryList[i] = t
			go entryProcessing(i, entryChan)
			log.Printf("gateList.SetUp: defined ENTRY %v as %v\n", i, entryList[i])
			i += 1
			data = os.Getenv("ENTRY_" + strconv.Itoa(i))
		}
	}
}
