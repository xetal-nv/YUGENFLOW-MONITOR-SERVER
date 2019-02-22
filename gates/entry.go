package gates

import (
	"countingserver/spaces"
	"countingserver/support"
	"fmt"
	"log"
)

func entryProcessing(id int, in chan sensorData) {
	var scratchPad scratchData
	sensorListEntry := make(map[int]sensorData)
	gateListEntry := entryList[id].gates

	scratchPad.senData = make(map[int]sensorData)
	scratchPad.unusedSampleSum = make(map[int]int)

	for i := range entryList[id].senDef {
		scratchPad.senData[i] = sensorData{i, 0, 0}
		sensorListEntry[i] = sensorData{i, 0, 0}
	}
	for i := range sensorListEntry {
		scratchPad.unusedSampleSum[i] = 0
	}

	entryProcessingCore(id, in, sensorListEntry, gateListEntry, scratchPad)

}

func entryProcessingCore(id int, in chan sensorData, sensorListEntry map[int]sensorData, gateListEntry map[int][]int, scratchPad scratchData) {
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				log.Printf("gates.entryProcessing: recovering for entry thread %v due to %v\n ", id, e)
				go entryProcessingCore(id, in, sensorListEntry, gateListEntry, scratchPad)
			}
		}
	}()
	log.Printf("gates.entry: Processing: setting entry %v\n", id)
	for {
		data := <-in
		nv := data.val
		if support.Debug != 2 && support.Debug != 4 {
			sensorListEntry[data.num] = data
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

// TODO tested in real office
// NOTE extend to more than 2 devices per gate
func trackPeople(id int, sensorListEntry map[int]sensorData, gateListEntry map[int][]int,
	scratchPad scratchData) (map[int]sensorData, map[int][]int, scratchData, int) {
	rt := 0
	// it might be needed to keep the flag in the scratchpad, to be tested
	flag := make([]bool, len(sensorListEntry), len(sensorListEntry))

	// NOTE it might have an issue with extremely fast noise ona  device faster than 1 ms
	for i, sen := range sensorListEntry {
		smem := scratchPad.senData[i]
		if smem.ts != sen.ts || smem.val != sen.val { //new sample detected
			smem.ts = sen.ts
			smem.val = sen.val
			scratchPad.senData[i] = smem
			scratchPad.unusedSampleSum[i] += sen.val
			flag[i] = true
		}
	}

	for _, gate := range gateListEntry {
		if scratchPad.unusedSampleSum[gate[0]] > 0 && scratchPad.unusedSampleSum[gate[1]] > 0 { //in
			tmp := support.Min(support.Abs(scratchPad.unusedSampleSum[gate[0]]),
				support.Abs(scratchPad.unusedSampleSum[gate[1]]))
			rt += tmp
			scratchPad.unusedSampleSum[gate[0]] -= tmp
			scratchPad.unusedSampleSum[gate[1]] -= tmp
		} else if scratchPad.unusedSampleSum[gate[0]] < 0 && scratchPad.unusedSampleSum[gate[1]] < 0 { //out
			tmp := support.Min(support.Abs(scratchPad.unusedSampleSum[gate[0]]),
				support.Abs(scratchPad.unusedSampleSum[gate[1]]))
			rt -= tmp
			scratchPad.unusedSampleSum[gate[0]] += tmp
			scratchPad.unusedSampleSum[gate[1]] += tmp
		}
	}

	for _, gate := range gateListEntry {
		// out not detected by sensor 1
		if scratchPad.unusedSampleSum[gate[0]] < 0 {
			scratchPad.unusedSampleSum[gate[0]] = 0
			rt--
		}
		// in not detected by sensor 0
		if scratchPad.unusedSampleSum[gate[1]] > 0 {
			scratchPad.unusedSampleSum[gate[1]] = 0
			rt++
		}
		// in not detected by sensor 1
		if flag[gate[1]] && scratchPad.senData[gate[1]].val == 0 && scratchPad.unusedSampleSum[gate[0]] > 0 {
			// if flag in the scratchPad it needs to ne reset
			rt++
			scratchPad.unusedSampleSum[gate[0]]--
		}
		// out not detected by sensor 0
		if flag[gate[0]] && scratchPad.senData[gate[0]].val == 0 && scratchPad.unusedSampleSum[gate[1]] < 0 {
			// if flag in the scratchPad it needs to ne reset
			rt--
			scratchPad.unusedSampleSum[gate[1]]++
		}
	}

	if support.Debug > 0 {
		//fmt.Printf("\nEntry %v has sensorListEntry:\n\t%v\n", id, sensorListEntry)
		//fmt.Printf("Entry %v has gateListEntry:\n\t%v\n", id, gateListEntry)
		fmt.Printf("Entry %v has scratchPad:\n\t%v\n", id, scratchPad)
	}

	return sensorListEntry, gateListEntry, scratchPad, rt
}
