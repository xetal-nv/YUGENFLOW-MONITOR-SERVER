package gates

import (
	"gateserver/support"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

// set-up for all sensor/gate/entry variables, threads and channels based on the data from the configuration file .env

func SetUp() {
	var revdev []int
	SensorRst.Lock()
	SensorRst.Channel = make(map[int]chan bool)
	SensorRst.Unlock()

	maximumAsymmetry = 0
	if v, e := strconv.Atoi(strings.Trim(os.Getenv("RESETPERIOD"), " ")); e == nil {
		if v != 0 {
			if da := os.Getenv("DEVICEASYM"); da == "" {
				maximumAsymmetry = 5
			} else {
				if v, e := strconv.Atoi(da); e != nil {
					log.Fatal("spaces.SetUp: fatal error in definition of DEVICEASYM")
				} else {
					maximumAsymmetry = v
				}
			}
		}
	}
	if maximumAsymmetry == 0 {
		log.Printf("!!! WARNING GATE ASYMMETRY RESET IS NOT ENABLED !!!\n")
	} else {
		if da := os.Getenv("DEVICEASYMCMAX"); da == "" {
			maximumAsymmetryIter = 4
		} else {
			if v, e := strconv.Atoi(da); e != nil {
				log.Fatal("spaces.SetUp: fatal error in definition of DEVICEASYMCMAX")
			} else {
				maximumAsymmetryIter = v
			}
		}
		log.Printf("!!! WARNING GATE ASYMMETRY RESET IS ENABLED !!!\n")
		log.Printf("spaces.SetUp: setting maximum asymmetry for gate sensors at %v with maximum iters at %v\n", maximumAsymmetry, maximumAsymmetryIter)
	}

	if data := os.Getenv("REVERSE"); data != "" {
		for _, v := range strings.Split(data, " ") {
			if vi, e := strconv.Atoi(v); e == nil {
				revdev = append(revdev, vi)
			} else {
				log.Fatal("gateList.SetUp: fatal error converting Reversed gate name ", v)
			}
		}
		log.Println("gateList.SetUp: defined Reversed Gates", revdev)
	}
	i := 0
	if data := os.Getenv("GATE_" + strconv.Itoa(i)); data == "" {
		log.Fatal("gateList.SetUp: fatal error, no gate has been defined")
	} else {
		sensorList = make(map[int]SensorDef)
		gateList = make(map[int][]int)
		DeclaredDevices = make(map[string]int)
		for data != "" {
			t := strings.Split(strings.Trim(data, " "), " ")
			if len(t) < 1 || len(t) > maxDevicePerGate {
				log.Fatal("gateList.SetUp: fatal error, illegal number of devices for gate ", i)
			}
			for _, v := range t {
				devdat := strings.Split(v, "?")
				if ind, ok := strconv.Atoi(devdat[len(devdat)-1]); ok != nil {
					log.Fatal("gateList.SetUp: fatal error in definition of GATE ", i)
				} else {
					if len(devdat) <= 2 {
						// add device declaration
						if len(devdat) == 2 {
							var mac []byte
							if c, e := net.ParseMAC(devdat[0]); e == nil {
								for _, v := range c {
									mac = append(mac, v)
								}
							}
							DeclaredDevices[string(mac)] = ind
						}
						// verify reverse status
						rev := false
						if support.Contains(revdev, ind) {
							rev = true
						}
						// add sensor to sensor list
						if v, ok := sensorList[ind]; ok {
							v.gate = append(v.gate, i)
							sensorList[ind] = v
						} else {
							sensorList[ind] = SensorDef{id: ind, Reversed: rev, gate: []int{i}}
						}

						gateList[i] = append(gateList[i], ind)
					}
				}
			}
			log.Printf("gateList.SetUp: defined gate %v as [Id Reversed]:\n", i)
			for _, v := range gateList[i] {
				log.Printf("\t\t [%v %v]\n", sensorList[v].id, sensorList[v].Reversed)
			}
			i += 1
			data = os.Getenv("GATE_" + strconv.Itoa(i))
		}
	}

	// removes possible replication of devices in a gate
	for i, gate := range gateList {
		var list []int
		keys := make(map[int]bool)
		for _, entry := range gate {
			if _, value := keys[entry]; !value {
				keys[entry] = true
				list = append(list, entry)
			}
		}
		gateList[i] = list
	}

	if data := os.Getenv("SPARES"); data != "" {
		for _, macst := range strings.Split(strings.Trim(data, " "), " ") {
			var mac []byte
			if c, e := net.ParseMAC(macst); e == nil {
				for _, v := range c {
					mac = append(mac, v)
				}
			}
			DeclaredDevices[macst] = 65535
		}
	}
	i = 0
	if data := os.Getenv("ENTRY_" + strconv.Itoa(i)); data == "" {
		log.Fatal("gateList.SetUp: fatal error, no entry has been defined")
	} else {
		EntryList = make(map[int]EntryDef)
		for data != "" {
			t := EntryDef{Id: 1}
			t.SenDef = make(map[int]SensorDef)
			t.Gates = make(map[int][]int)
			entryChan := make(chan sensorData)
			for _, v := range strings.Split(strings.Trim(data, " "), " ") {
				if ind, ok := strconv.Atoi(v); ok != nil {
					log.Fatal("gateList.SetUp: fatal error in definition of ENTRY ", i)
				} else {
					t.Gates[ind] = gateList[ind]
					for _, d := range gateList[ind] {
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
						t.SenDef[d] = sensorList[d]
					}
				}
			}
			EntryList[i] = t
			go entryProcessingSetUp(i, entryChan, t)
			log.Printf("gateList.SetUp: defined ENTRY %v as %v\n", i, t)
			i += 1
			data = os.Getenv("ENTRY_" + strconv.Itoa(i))
		}
	}
}
