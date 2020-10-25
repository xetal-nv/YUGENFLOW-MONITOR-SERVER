package gateManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/entryManager"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"math"
	"os"
	"strconv"
	"time"
)

// NOTE: the code is build for one or two sensors gates.
func detectTransition(id string, gateSensorsOrdered []int, sensorLatestData map[int]sensorData,
	scratchPad scratchData) (map[int]sensorData, scratchData, int) {
	rt := 0
	flag := make(map[int]bool)
	for i := range sensorLatestData {
		flag[i] = false
	}

	// get new samples and clean scratchpad from not allowed pos and negs
	for i, sensor := range sensorLatestData {
		scratchPadSensor := scratchPad.senData[i]
		// if the timestamp is the same it is the impossible situation of simultaneous arrival (at 1ns level) and we look at the value change
		if scratchPadSensor.ts != sensor.ts || scratchPadSensor.val != sensor.val {
			//new sample detected, the scratchpad is updated
			scratchPadSensor.ts = sensor.ts
			scratchPadSensor.val = sensor.val
			scratchPad.senData[i] = scratchPadSensor
			scratchPad.unusedSampleSumIn[i] += sensor.val
			scratchPad.unusedSampleSumOut[i] += sensor.val
			if scratchPad.unusedSampleSumIn[i] < 0 {
				scratchPad.unusedSampleSumIn[i] = 0
			}
			if scratchPad.unusedSampleSumOut[i] > 0 {
				scratchPad.unusedSampleSumOut[i] = 0
			}
			flag[i] = true
		}
	}

	if len(gateSensorsOrdered) == 1 {
		// in case of single device the data is passed as it
		rt = scratchPad.senData[gateSensorsOrdered[0]].val
		scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]] = 0
		scratchPad.unusedSampleSumOut[gateSensorsOrdered[0]] = 0
	} else {
		if scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]] > 0 && scratchPad.unusedSampleSumIn[gateSensorsOrdered[1]] > 0 {
			//in
			tmp := others.Min(others.Abs(scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]]),
				others.Abs(scratchPad.unusedSampleSumIn[gateSensorsOrdered[1]]))
			rt += tmp
			scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]] -= tmp
			scratchPad.unusedSampleSumIn[gateSensorsOrdered[1]] -= tmp
			if scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]] < 0 {
				scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]] = 0
			}
			if scratchPad.unusedSampleSumIn[gateSensorsOrdered[1]] < 0 {
				scratchPad.unusedSampleSumIn[gateSensorsOrdered[1]] = 0
			}
		}
		if scratchPad.unusedSampleSumOut[gateSensorsOrdered[0]] < 0 && scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]] < 0 {
			//out
			tmp := others.Min(others.Abs(scratchPad.unusedSampleSumOut[gateSensorsOrdered[0]]),
				others.Abs(scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]]))
			rt -= tmp
			scratchPad.unusedSampleSumOut[gateSensorsOrdered[0]] += tmp
			scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]] += tmp
			if scratchPad.unusedSampleSumOut[gateSensorsOrdered[0]] > 0 {
				scratchPad.unusedSampleSumOut[gateSensorsOrdered[0]] = 0
			}
			if scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]] > 0 {
				scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]] = 0
			}
		}
	}

	if len(gateSensorsOrdered) > 1 {
		// in - not detected by sensor 1
		if flag[gateSensorsOrdered[1]] && scratchPad.senData[gateSensorsOrdered[1]].val == 0 && scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]] > 0 {
			// if flag in the scratchPad it needs to be reset
			rt++
			scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]]--
		}
		// out - not detected by sensor 0
		if flag[gateSensorsOrdered[0]] && scratchPad.senData[gateSensorsOrdered[0]].val == 0 && scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]] < 0 {
			// if flag in the scratchPad it needs to be reset
			rt--
			scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]]++
		}

		// cleaning in case or large asymmetries due to defected sensor
		if scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]] > 2 {
			rt += 1
			scratchPad.unusedSampleSumIn[gateSensorsOrdered[0]] -= 1
		}
		if scratchPad.unusedSampleSumIn[gateSensorsOrdered[1]] > 2 {
			rt += 1
			scratchPad.unusedSampleSumIn[gateSensorsOrdered[1]] -= 1
		}
		if scratchPad.unusedSampleSumOut[gateSensorsOrdered[0]] < -2 {
			rt -= 1
			scratchPad.unusedSampleSumOut[gateSensorsOrdered[0]] += 1
		}
		if scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]] < -2 {
			rt -= 1
			scratchPad.unusedSampleSumOut[gateSensorsOrdered[1]] += 1
		}
	}

	if globals.DebugActive {
		fmt.Printf("Gate %v has scratchPad:\n\t%+v\n", id, scratchPad)
	}

	return sensorLatestData, scratchPad, rt
}

func gate(gateName string, gateSensorsOrdered []int, in chan dataformats.FlowData, stop chan interface{},
	resetGate chan interface{}, sensors map[int]dataformats.SensorDefinition) {

	var scratchPad, scratchPadOriginal scratchData
	gateSensorsOrderedOriginal := make([]int, len(gateSensorsOrdered))
	sensorDifferential := make(map[int]int)
	//sensorDifferentialFails := make(map[int]int)
	sensorDifferentialTimes := make(map[int]int)
	sensorLatestData := make(map[int]sensorData)
	sensorLatestDataOriginal := make(map[int]sensorData)
	scratchPad.senData = make(map[int]sensorData)
	scratchPad.unusedSampleSumIn = make(map[int]int)
	scratchPad.unusedSampleSumOut = make(map[int]int)
	scratchPadOriginal.senData = make(map[int]sensorData)
	scratchPadOriginal.unusedSampleSumIn = make(map[int]int)
	scratchPadOriginal.unusedSampleSumOut = make(map[int]int)
	copy(gateSensorsOrderedOriginal, gateSensorsOrdered)

	asymmetryResetCounter := globals.AsymmetryReset

	for i := range sensors {
		scratchPad.senData[i] = sensorData{i, 0, 0}
		scratchPadOriginal.senData[i] = sensorData{i, 0, 0}
		sensorLatestData[i] = sensorData{i, 0, 0}
		sensorLatestDataOriginal[i] = sensorData{i, 0, 0}
		sensorDifferential[i] = 0
		//sensorDifferentialFails[i] = 0
		sensorDifferentialTimes[i] = 0
	}
	for i := range sensorLatestData {
		scratchPad.unusedSampleSumIn[i] = 0
		scratchPad.unusedSampleSumOut[i] = 0
		scratchPadOriginal.unusedSampleSumIn[i] = 0
		scratchPadOriginal.unusedSampleSumOut[i] = 0
	}

	//fmt.Println(scratchPad)
	//fmt.Println(sensorLatestData)

	// we try to acquire the gate channels
	// this will only fail if the server is corrupted therefore, the server should be closed
	tries := 5
	day := time.Now().Day()
	entryManager.GateStructure.RLock()
	gateEntryChannels, ok := entryManager.GateStructure.DataChannel[gateName]
	entryManager.GateStructure.RUnlock()
	for !ok {
		if tries == 0 {
			fmt.Printf("Gate %v has failed to start or is not used\n", gateName)
			os.Exit(0)
		} else {
			tries -= 1
		}
		time.Sleep(time.Duration(globals.SettleTime) * time.Second)
		entryManager.GateStructure.RLock()
		gateEntryChannels, ok = entryManager.GateStructure.DataChannel[gateName]
		entryManager.GateStructure.RUnlock()
	}

	//fmt.Println(gateName, gateEntryChannels)

	if globals.DebugActive {
		fmt.Printf("Gate %v has been started\n", gateName)
	}
	//fmt.Println(in, stop, gateName, sensors)
	for {
		select {
		case <-resetGate:
			// the gate configuration is reset
			asymmetryResetCounter = globals.AsymmetryReset
			sensorDifferential = make(map[int]int)
			//sensorDifferentialFails = make(map[int]int)
			sensorDifferentialTimes = make(map[int]int)
			gateSensorsOrdered = make([]int, len(gateSensorsOrderedOriginal))
			copy(gateSensorsOrdered, gateSensorsOrderedOriginal)
			scratchPad.senData = make(map[int]sensorData)
			scratchPad.unusedSampleSumIn = make(map[int]int)
			scratchPad.unusedSampleSumOut = make(map[int]int)
			for i, el := range scratchPadOriginal.senData {
				scratchPad.senData[i] = el
			}
			for i, el := range scratchPadOriginal.unusedSampleSumOut {
				scratchPad.unusedSampleSumOut[i] = el
				sensorDifferential[i] = 0
				//sensorDifferentialFails[i] = 0
				sensorDifferentialTimes[i] = 0
			}
			for i, el := range scratchPadOriginal.unusedSampleSumIn {
				scratchPad.unusedSampleSumIn[i] = el
			}
			sensorLatestData = make(map[int]sensorData)
			for i, el := range sensorLatestDataOriginal {
				sensorLatestData[i] = el
			}
			if globals.DebugActive {
				fmt.Printf("gateManager.gate: gate %v configuration reset\n", gateName)
			}
			mlogger.Info(globals.GateManagerLog,
				mlogger.LoggerData{"gateManager.gate: " + gateName,
					"gate configuration reset",
					[]int{0}, true})

		case data := <-in:
			if _, ok := sensorLatestData[data.Id]; ok {
				if sensors[data.Id].Reversed {
					data.Netflow *= -1
				}
				if globals.DebugActive {
					fmt.Printf(" ===>>> Gate %v received: %+v\n", gateName, data)
				}
				if globals.AsymmetryIter != 0 && len(gateSensorsOrdered) > 1 &&
					!(globals.AsymmetricNull && data.Netflow == 0) {
					// every day we reset the asymmetry conditions
					if time.Now().Day() != day {
						day = time.Now().Day()
						for i := range sensorDifferential {
							sensorDifferential[i] = 0
						}
					}
					sensorDifferential[data.Id] += 1
					//fmt.Println(gateName, ":", sensorDifferential)
					var sensorID int
					max := 0
					min := math.MaxInt32
					for i, e := range sensorDifferential {
						if e < min {
							min = e
							sensorID = i
						}
					}
					for i, e := range sensorDifferential {
						if e > max {
							max = e
						}
						// avoid the numbers gets too large
						sensorDifferential[i] -= min
					}
					if max-min >= globals.AsymmetryMax {
						sensorDifferentialTimes[sensorID] += 1
						if sensorDifferentialTimes[sensorID] < globals.AsymmetryIter {
							// we try to reset the background of all sensors in the gate
							for id := range sensorDifferential {
								sensorDifferential[id] = 0
								if mac, err := diskCache.LookUpMac([]byte{byte(id)}); err == nil {
									select {
									case globals.ResetChannel <- string(mac):
									case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
										if globals.DebugActive {
											fmt.Printf("gateManager.gate: gate %v sensor %v reset failed\n", gateName, id)
										}
										mlogger.Warning(globals.GateManagerLog,
											mlogger.LoggerData{"sensorManager.gateManager: " + strconv.Itoa(id),
												"reset has failed for timeout",
												[]int{1}, true})
									}
								} else {
									if globals.DebugActive {
										fmt.Printf("gateManager.gate: gate %v sensor %v not properly active\n", gateName, id)
									}
									mlogger.Warning(globals.GateManagerLog,
										mlogger.LoggerData{"sensorManager.gateManager: " + strconv.Itoa(id),
											"reset has failed for missing entry",
											[]int{1}, true})
								}
							}
						} else {
							if globals.DebugActive {
								fmt.Printf("gateManager.gate: gate %v sensor %v disabled\n", gateName, sensorID)
							}
							mlogger.Warning(globals.GateManagerLog,
								mlogger.LoggerData{"gateManager.gate: " + gateName,
									"sensor disabled: " + strconv.Itoa(sensorID),
									[]int{1}, true})
							delete(sensorLatestData, sensorID)
							delete(scratchPad.senData, sensorID)
							delete(scratchPad.unusedSampleSumIn, sensorID)
							delete(scratchPad.unusedSampleSumOut, sensorID)
							gateSensorsOrdered = nil
							for i := range sensorLatestData {
								gateSensorsOrdered = append(gateSensorsOrdered, i)
							}
							for i := range sensorDifferential {
								sensorDifferential[i] = 0
							}
						}
					}
				}

				//fmt.Println(gateSensorsOrdered, sensorLatestData, scratchPad)
				sensorLatestData[data.Id] = sensorData{
					id:  data.Id,
					ts:  data.Ts,
					val: data.Netflow,
				}
				var nv int
				sensorLatestData, scratchPad, nv = detectTransition(gateName, gateSensorsOrdered, sensorLatestData, scratchPad)
				newGateData := dataformats.FlowData{
					Type:    "gate",
					Name:    gateName,
					Id:      0,
					Ts:      data.Ts,
					Netflow: nv,
				}
				if globals.DebugActive {
					fmt.Printf(" ===>>> Gate %v calculated value: %+v\n", gateName, nv)
				}
				// after a globals.AsymmetryReset number of correct samples asymmetry is reset
				if asymmetryResetCounter -= 1; asymmetryResetCounter == 0 {
					for i, el := range scratchPadOriginal.unusedSampleSumOut {
						scratchPad.unusedSampleSumOut[i] = el
						sensorDifferential[i] = 0
						//sensorDifferentialFails[i] = 0
						sensorDifferentialTimes[i] = 0
						asymmetryResetCounter = globals.AsymmetryReset
					}
				}
				for _, ch := range gateEntryChannels {
					ch <- newGateData
				}
			} else {
				if globals.DebugActive {
					fmt.Printf(" ===>>> Gate %v reject: %+v\n", gateName, data)
				}
			}

		case <-stop:
			if globals.DebugActive {
				fmt.Printf("Gate %v has been stopped\n", gateName)
			}
			stop <- nil
		}
	}
}
