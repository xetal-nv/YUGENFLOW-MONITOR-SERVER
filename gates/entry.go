package gates

import (
	"fmt"
	"gateserver/spaces"
	"gateserver/support"
	"log"
	"os"
	"strconv"
	"sync"
)

// set-up for the processing of sensor/gate data into flow values for the associated entry id
// new sensor data is passed by means of the in channel snd send to the proper space via a spaces.SendData call
func entryProcessingSetUp(id int, in chan sensorData, entryList EntryDef) {
	var scratchPad scratchData
	sensorListEntry := make(map[int]sensorData)
	sensorToGate := make(map[int]int)
	sensorSensorDifferential := make(map[int][]int)
	//gateListEntry := entryList.Gates

	scratchPad.senData = make(map[int]sensorData)
	scratchPad.unusedSampleSumIn = make(map[int]int)
	scratchPad.unusedSampleSumOut = make(map[int]int)

	//for i := range EntryList[id].SenDef {
	for i := range entryList.SenDef {
		scratchPad.senData[i] = sensorData{i, 0, 0}
		sensorListEntry[i] = sensorData{i, 0, 0}
	}
	for i := range sensorListEntry {
		scratchPad.unusedSampleSumIn[i] = 0
		scratchPad.unusedSampleSumOut[i] = 0
	}
	for ind, el := range entryList.Gates {
		for _, sen := range el {
			sensorToGate[sen] = ind
			sensorSensorDifferential[ind] = append(sensorSensorDifferential[ind], 0)
		}
	}

	entryProcessingCore(id, in, sensorListEntry, entryList.Gates, scratchPad, sensorToGate, sensorSensorDifferential)
}

// implements the core logic od the sensor/gate data processing
func entryProcessingCore(id int, in chan sensorData, sensorListEntry map[int]sensorData,
	gateListEntry map[int][]int, scratchPad scratchData, sensorToGate map[int]int, sensorSensorDifferential map[int][]int) {
	var f *os.File
	var err error
	var tryResetMux sync.RWMutex
	tryReset := make(map[int]bool)
	if LogToFileAll {
		f, err = os.Create("log/entry_" + strconv.Itoa(id) + ".txt")
	}
	for senId := range sensorToGate {
		tryReset[senId] = true
	}

	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"Gates.entryProcessingCore",
					support.Timestamp(), "", []int{1}, true}
			}()
			if err == nil && LogToFileAll {
				_ = f.Close()
			}
			log.Printf("Gates.entryProcessingCore: recovering for entry %v due to %v\n ", id, e)
			go entryProcessingCore(id, in, sensorListEntry, gateListEntry, scratchPad, sensorToGate, sensorSensorDifferential)
		}
	}()
	log.Printf("Gates.entry: Processing: setting entry %v\n", id)
	for {
		data := <-in
		nv := data.val
		// check for asymmetry in gate sensor dictating reset
		for i, r := range gateListEntry[sensorToGate[data.id]] {
			if r == data.id {
				//fmt.Println(id, sensorSensorDifferential)
				// if there is no pending reset request, check for a need for reset
				tryResetMux.RLock()
				if tryReset[r] {
					// this array does not need locking, races are not possible despite the go routine on it
					sensorSensorDifferential[sensorToGate[r]][i] += 1
					tryResetMux.RUnlock()
					var min, max int
					for i, e := range sensorSensorDifferential[sensorToGate[r]] {
						if i == 0 || e < min {
							min = e
						}
					}
					for i, e := range sensorSensorDifferential[sensorToGate[r]] {
						if i == 0 || e > max {
							max = e
						}
						sensorSensorDifferential[sensorToGate[r]][i] -= min
					}
					//fmt.Println(sensorToGate[r], max, min, r)
					if max-min >= maximumAsymmetry {
						//tryReset.Lock() // this locking is redundant
						tryReset[r] = false
						//tryReset.Unlock() // this locking is redundant
						go func(id int) {
							// we do not wait for a success as failure to execute will eventually result in another reset request
							SensorRst.RLock()
							resetChannel, ok := SensorRst.Channel[id]
							SensorRst.RUnlock()
							if ok {
								resetChannel <- true
								//fmt.Println("sent reset request for id", id)
							}
							// ok false means that the self reset was not enabled
							for i := range sensorSensorDifferential[sensorToGate[id]] {
								sensorSensorDifferential[sensorToGate[id]][i] = 0
							}
							// for safety of races we should use a RW lock on tryReset
							tryResetMux.Lock()
							tryReset[r] = true
							tryResetMux.Unlock()
						}(r)
					}
				} else {
					tryResetMux.RUnlock()
					//fmt.Println("reset request pending for gate", sensorToGate[r])
				}
				break
			}
		}
		// calculates the next sample
		if support.Debug != 2 && support.Debug != 4 && support.Debug != -1 {
			sensorListEntry[data.id] = data
			sensorListEntry, gateListEntry, scratchPad, nv = trackPeople(id, sensorListEntry, gateListEntry, scratchPad)
		}
		if LogToFileAll {
			if err == nil {
				_, _ = f.WriteString("New sample\n")
				_, _ = f.WriteString("sensor data: ")
				for key, val := range scratchPad.senData {
					_, _ = f.WriteString("( " + strconv.Itoa(key) + "," + strconv.Itoa(int(val.ts)) + "," + strconv.Itoa(val.val) + " ) ")
				}
				_, _ = f.WriteString("\n")

				_, _ = f.WriteString("unusedSampleSumIn: ")
				for key, val := range scratchPad.unusedSampleSumIn {
					_, _ = f.WriteString("( " + strconv.Itoa(key) + "," + strconv.Itoa(val) + " ) ")
				}
				_, _ = f.WriteString("\n")

				_, _ = f.WriteString("unusedSampleSumOut: ")
				for key, val := range scratchPad.unusedSampleSumOut {
					_, _ = f.WriteString("( " + strconv.Itoa(key) + "," + strconv.Itoa(val) + " ) ")
				}
				_, _ = f.WriteString("\n")
				_, _ = f.WriteString("calculated datapoint: ")
				_, _ = f.WriteString("( " + strconv.Itoa(int(support.Timestamp())) + "," + strconv.Itoa(nv) + " ) ")

				_, _ = f.WriteString("\n\n")
			}
		}
		if e := spaces.SendData(id, nv); e != nil {
			log.Println(e)
		}
		if support.Debug != 0 {
			fmt.Printf("\nEntry %v has calculated datapoint at %v as %v\n", id, support.Timestamp(), nv)
		}
	}

}

// implements the algorithm logic od the gate data processing
func trackPeople(id int, sensorListEntry map[int]sensorData, gateListEntry map[int][]int,
	scratchPad scratchData) (map[int]sensorData, map[int][]int, scratchData, int) {
	rt := 0
	flag := make(map[int]bool)
	for i := range sensorListEntry {
		flag[i] = false
	}

	// get new samples and clean scratchpad from not allowed pos and negs
	for i, sen := range sensorListEntry {
		smem := scratchPad.senData[i]
		if smem.ts != sen.ts || smem.val != sen.val { //new sample detected
			smem.ts = sen.ts
			smem.val = sen.val
			scratchPad.senData[i] = smem
			scratchPad.unusedSampleSumIn[i] += sen.val
			scratchPad.unusedSampleSumOut[i] += sen.val
			if scratchPad.unusedSampleSumIn[i] < 0 {
				scratchPad.unusedSampleSumIn[i] = 0
			}
			if scratchPad.unusedSampleSumOut[i] > 0 {
				scratchPad.unusedSampleSumOut[i] = 0
			}
			flag[i] = true
		}
	}

	for _, gate := range gateListEntry {
		if scratchPad.unusedSampleSumIn[gate[0]] > 0 && scratchPad.unusedSampleSumIn[gate[1]] > 0 { //in
			tmp := support.Min(support.Abs(scratchPad.unusedSampleSumIn[gate[0]]),
				support.Abs(scratchPad.unusedSampleSumIn[gate[1]]))
			rt += tmp
			scratchPad.unusedSampleSumIn[gate[0]] -= tmp
			scratchPad.unusedSampleSumIn[gate[1]] -= tmp
			if scratchPad.unusedSampleSumIn[gate[0]] < 0 {
				scratchPad.unusedSampleSumIn[gate[0]] = 0
			}
			if scratchPad.unusedSampleSumIn[gate[1]] < 0 {
				scratchPad.unusedSampleSumIn[gate[1]] = 0
			}
		}
		if scratchPad.unusedSampleSumOut[gate[0]] < 0 && scratchPad.unusedSampleSumOut[gate[1]] < 0 { //out
			tmp := support.Min(support.Abs(scratchPad.unusedSampleSumOut[gate[0]]),
				support.Abs(scratchPad.unusedSampleSumOut[gate[1]]))
			rt -= tmp
			scratchPad.unusedSampleSumOut[gate[0]] += tmp
			scratchPad.unusedSampleSumOut[gate[1]] += tmp
			if scratchPad.unusedSampleSumOut[gate[0]] > 0 {
				scratchPad.unusedSampleSumOut[gate[0]] = 0
			}
			if scratchPad.unusedSampleSumOut[gate[1]] > 0 {
				scratchPad.unusedSampleSumOut[gate[1]] = 0
			}
		}

	}

	for _, gate := range gateListEntry {
		// in - not detected by sensor 1
		if flag[gate[1]] && scratchPad.senData[gate[1]].val == 0 && scratchPad.unusedSampleSumIn[gate[0]] > 0 {
			// if flag in the scratchPad it needs to be reset
			rt++
			scratchPad.unusedSampleSumIn[gate[0]]--
		}
		// out - not detected by sensor 0
		if flag[gate[0]] && scratchPad.senData[gate[0]].val == 0 && scratchPad.unusedSampleSumOut[gate[1]] < 0 {
			// if flag in the scratchPad it needs to be reset
			rt--
			scratchPad.unusedSampleSumOut[gate[1]]++
		}

		// cleaning in case or large asymmetries due to defected sensor
		if scratchPad.unusedSampleSumIn[gate[0]] > 2 {
			rt += 1
			scratchPad.unusedSampleSumIn[gate[0]] -= 1
		}
		if scratchPad.unusedSampleSumIn[gate[1]] > 2 {
			rt += 1
			scratchPad.unusedSampleSumIn[gate[1]] -= 1
		}
		if scratchPad.unusedSampleSumOut[gate[0]] < -2 {
			rt -= 1
			scratchPad.unusedSampleSumOut[gate[0]] += 1
		}
		if scratchPad.unusedSampleSumOut[gate[1]] < -2 {
			rt -= 1
			scratchPad.unusedSampleSumOut[gate[1]] += 1
		}
	}

	if support.Debug > 0 {
		//fmt.Printf("\nEntry %v has sensorListEntry:\n\t%v\n", Id, sensorListEntry)
		//fmt.Printf("Entry %v has gateListEntry:\n\t%v\n", Id, gateListEntry)
		fmt.Printf("Entry %v has scratchPad:\n\t%v\n", id, scratchPad)
	}

	return sensorListEntry, gateListEntry, scratchPad, rt
}
