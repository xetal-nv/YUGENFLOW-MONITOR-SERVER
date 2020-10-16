package gateManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"strconv"
	"strings"
	"time"
	"xetal.ddns.net/utils/recovery"
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

	SensorStructure.Lock()
	GateStructure.Lock()
	SensorStructure.GateList = make(map[int][]string)
	SensorStructure.DataChannel = make(map[int]([]chan dataformats.FlowData))
	GateStructure.SensorList = make(map[string]map[int]dataformats.SensorDefinition)
	GateStructure.DataChannel = make(map[string]chan dataformats.FlowData)
	GateStructure.ConfigurationReset = make(map[string]chan interface{})
	GateStructure.StopChannel = make(map[string]chan interface{})

	for _, gt := range globals.Config.Section("gates").KeyStrings() {
		currentGate := gt
		if _, ok := GateStructure.SensorList[currentGate]; ok {
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
						SensorStructure.GateList[sensorId] = append(SensorStructure.GateList[sensorId], currentGate)
						SensorStructure.DataChannel[sensorId] = append(SensorStructure.DataChannel[sensorId], newDataChannel)
						if GateStructure.SensorList[currentGate] == nil {
							GateStructure.SensorList[currentGate] = make(map[int]dataformats.SensorDefinition)
						}
						GateStructure.SensorList[currentGate][sensorId] = dataformats.SensorDefinition{
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
				if GateStructure.SensorList[currentGate] != nil {
					copyMap := make(map[int]dataformats.SensorDefinition)
					for k, v := range GateStructure.SensorList[currentGate] {
						copyMap[k] = v
					}
					GateStructure.DataChannel[currentGate] = newDataChannel
					chanReset := make(chan interface{}, 1)
					GateStructure.ConfigurationReset[currentGate] = chanReset
					chanStop := make(chan interface{}, 1)
					GateStructure.StopChannel[currentGate] = chanStop
					go recovery.RunWith(
						func() {
							gate(currentGate, gateSensorsOrdered, newDataChannel, chanStop, chanReset, copyMap)
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

	GateStructure.Unlock()
	SensorStructure.Unlock()

	// shutdown procedure
	<-sd
	fmt.Println("Closing gateManager")
	//var wg sync.WaitGroup
	GateStructure.Lock()
	for _, ch := range GateStructure.StopChannel {
		//wg.Add(1)
		//go func(ch chan interface{}) {
		ch <- nil
		select {
		case <-ch:
		case <-time.After(time.Duration(globals.SettleTime) * time.Second):
		}
		//wg.Done()
		//}(ch)
	}
	GateStructure.Unlock()
	//wg.Wait()
	mlogger.Info(globals.GateManagerLog,
		mlogger.LoggerData{"gateManager.Start",
			"service stopped",
			[]int{0}, true})
	time.Sleep(time.Duration(globals.SettleTime) * time.Second)
	sd <- true
}
