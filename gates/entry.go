package gates

import (
	"fmt"
	"gateserver/spaces"
	"gateserver/support"
	"log"
)

// set-up for the processing of sensor/gate data into flow values for the associated entry id
// new sensor data is passed by means of the in channel snd send to the proper space via a spaces.SendData call
func entryProcessingSetUp(id int, in chan sensorData, entrylist EntryDef) {
	var scratchPad scratchData
	sensorListEntry := make(map[int]sensorData)
	gateListEntry := entrylist.Gates

	scratchPad.senData = make(map[int]sensorData)
	scratchPad.unusedSampleSumIn = make(map[int]int)
	scratchPad.unusedSampleSumOut = make(map[int]int)

	//for i := range EntryList[id].SenDef {
	for i := range entrylist.SenDef {
		scratchPad.senData[i] = sensorData{i, 0, 0}
		sensorListEntry[i] = sensorData{i, 0, 0}
	}
	for i := range sensorListEntry {
		scratchPad.unusedSampleSumIn[i] = 0
		scratchPad.unusedSampleSumOut[i] = 0
	}
	entryProcessingCore(id, in, sensorListEntry, gateListEntry, scratchPad)

}

// implements the core logic od the sensor/gate data processing
func entryProcessingCore(id int, in chan sensorData, sensorListEntry map[int]sensorData,
	gateListEntry map[int][]int, scratchPad scratchData) {
	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"Gates.entryProcessingCore",
					support.Timestamp(), "", []int{1}, true}
			}()
			log.Printf("Gates.entryProcessingCore: recovering for entry %v due to %v\n ", id, e)
			go entryProcessingCore(id, in, sensorListEntry, gateListEntry, scratchPad)
		}
	}()
	log.Printf("Gates.entry: Processing: setting entry %v\n", id)
	for {
		data := <-in
		nv := data.val
		if support.Debug != 2 && support.Debug != 4 && support.Debug != -1 {
			sensorListEntry[data.id] = data
			sensorListEntry, gateListEntry, scratchPad, nv = trackPeople(id, sensorListEntry, gateListEntry, scratchPad)
		}
		if e := spaces.SendData(id, nv); e != nil {
			log.Println(e)
		}
		if support.Debug > 0 {
			fmt.Printf("\nEntry %v has calculated datapoint at %v as %v\n", id, support.Timestamp(), nv)
		}
	}

}

// implements the algorithm logic od the gate data processing
// TODO extend to more than 2 devices per gate
func trackPeople(id int, sensorListEntry map[int]sensorData, gateListEntry map[int][]int,
	scratchPad scratchData) (map[int]sensorData, map[int][]int, scratchData, int) {
	rt := 0
	flag := make(map[int]bool)
	for i := range sensorListEntry {
		flag[i] = false
	}

	// NOTE it might have an issue with noise or a device faster than 1ms

	// bget new samples and clean scratchpad from not allowed pos and negs
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

		// TODO cleaning in case or large asymmetries due to defected sensor
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
