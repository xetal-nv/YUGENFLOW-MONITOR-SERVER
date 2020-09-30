package gateManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"yfserver/support/recovery"
)

/*
	set up the sensor2gates channels and relative gate services
	send data to the relevant entries
	save gate data in the database
*/

func Start(sd chan bool) {
	var err error

	if globals.GateManagerLog, err = mlogger.DeclareLog("yugenflow_gateManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_gateManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.GateManagerLog, 40, 30, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.GateManagerLog,
		mlogger.LoggerData{"gateManager.Start",
			"service started",
			[]int{0}, true})

	var rstC []chan interface{}
	for i := 0; i < 0; i++ {
		rstC = append(rstC, make(chan interface{}))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan interface{}) {
		<-sd
		fmt.Println("Closing gateManager")
		var wg sync.WaitGroup
		for _, ch := range rstC {
			wg.Add(1)
			go func(ch chan interface{}) {
				ch <- nil
				select {
				case <-ch:
				case <-time.After(2 * time.Second):
				}
				wg.Done()
			}(ch)
		}
		GateList.Lock()
		for _, ch := range GateList.StopChannel {
			wg.Add(1)
			go func(ch chan interface{}) {
				ch <- nil
				select {
				case <-ch:
				case <-time.After(2 * time.Second):
				}
				wg.Done()
			}(ch)
		}
		GateList.Unlock()
		wg.Wait()
		mlogger.Info(globals.GateManagerLog,
			mlogger.LoggerData{"gateManager.Start",
				"service stopped",
				[]int{0}, true})
		time.Sleep(time.Duration(globals.ShutdownTime) * time.Second)
		sd <- true
	}(sd, rstC)

	// initialisation of gates

	SensorList.Lock()
	GateList.Lock()
	SensorList.GateList = make(map[int][]string)
	SensorList.DataChannel = make(map[int]([]chan dataformats.FlowData))
	GateList.SensorList = make(map[string]map[int]dataformats.SensorDefinition)
	GateList.DataChannel = make(map[string]chan dataformats.FlowData)
	GateList.ConfigurationReset = make(map[string]chan interface{})
	GateList.StopChannel = make(map[string]chan interface{})

	for _, gt := range globals.Config.Section("gates").KeyStrings() {
		currentGate := gt
		if _, ok := GateList.SensorList[currentGate]; ok {
			fmt.Println("Duplicated gate %v in configuration.ini ignored\n", currentGate)
		} else {
			gateDef := globals.Config.Section("gates").Key(currentGate).MustString("")
			if gateDef != "" {
				var gateSensorsOrdered []int
				sensors := strings.Split(gateDef, " ")

				// semantics check of the gate definitions (reject sensor and !sensor in the same gate)
				illegal := false
				if len(sensors) == 2 {
					for i, s := range sensors {
						sensors[i] = strings.Trim(strings.Replace(s, "!", "", -1), "")
						if val, err := strconv.Atoi(sensors[i]); err == nil {
							gateSensorsOrdered = append(gateSensorsOrdered, val)
						} else {
							illegal = true
							break
						}
					}
					if !illegal {
						for _, s := range sensors {
							illegal = strings.Contains(" "+gateDef, " "+s) && strings.Contains(gateDef, "!"+s)
							if illegal {
								break
							}
						}
					}
				} else {
					illegal = true
				}
				if illegal {
					fmt.Printf("Invalid gate definition \"%v : %v\" in configuration.ini ignored.\n", currentGate, gateDef)
					continue
				}
				sensors = strings.Split(gateDef, " ")
				newDataChannel := make(chan dataformats.FlowData, globals.ChannellingLength)
				for _, currentSensor := range sensors {
					if sensorId, err := strconv.Atoi(strings.Trim(strings.Replace(currentSensor, "!", "", -1), "")); err == nil {
						SensorList.GateList[sensorId] = append(SensorList.GateList[sensorId], currentGate)
						SensorList.DataChannel[sensorId] = append(SensorList.DataChannel[sensorId], newDataChannel)
						if GateList.SensorList[currentGate] == nil {
							GateList.SensorList[currentGate] = make(map[int]dataformats.SensorDefinition)
						}
						GateList.SensorList[currentGate][sensorId] = dataformats.SensorDefinition{
							Id:        sensorId,
							Reversed:  strings.Contains(currentSensor, "!"),
							Suspected: 0,
							Disabled:  false,
						}
					} else {
						fmt.Printf("Invalid sensor definition %v in configuration.ini %v ignored\n", currentSensor, currentGate)
					}
				}
				// channels are created only if the sensor list is valid
				if GateList.SensorList[currentGate] != nil {
					GateList.DataChannel[currentGate] = newDataChannel
					GateList.ConfigurationReset[currentGate] = make(chan interface{}, 1)
					GateList.StopChannel[currentGate] = make(chan interface{}, 1)
					go recovery.RunWith(
						func() {
							gate(currentGate, gateSensorsOrdered, GateList.DataChannel[currentGate], GateList.StopChannel[currentGate],
								GateList.ConfigurationReset[currentGate], GateList.SensorList[currentGate])
						},
						func() {
							mlogger.Recovered(globals.GateManagerLog,
								mlogger.LoggerData{"gateManager.gate: " + currentGate,
									"service terminated and recovered unexpectedly",
									[]int{1}, true})
						})
				}
			} else {
				fmt.Printf("Invalid gate definition %v in configuration.ini ignored\n", currentGate)
			}
		}
	}

	//fmt.Printf("%+v", GateList.SensorList)
	GateList.Unlock()
	SensorList.Unlock()
	//os.Exit(0)

	//for _, current := range globals.Config.Section("entries").KeyStrings() {
	//	fmt.Println(current, globals.Config.Section("entries").Key(current))
	//}
	//
	//for _, current := range globals.Config.Section("spaces").KeyStrings() {
	//	fmt.Println(current, globals.Config.Section("spaces").Key(current))
	//}

	//recovery.RunWith(
	//	func() { ApiManager(rstC[0]) },
	//	func() {
	//		mlogger.Recovered(globals.GateManagerLog,
	//			mlogger.LoggerData{"clientManager.ApiManager",
	//				"ApiManager service terminated and recovered unexpectedly",
	//				[]int{1}, true})
	//	})

	for {
		time.Sleep(36 * time.Hour)
	}
}
