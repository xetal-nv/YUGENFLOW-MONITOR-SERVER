package gateManager

// sensor data as identified id, timestamp ts and value val
type sensorData struct {
	id  int   // sensor number
	ts  int64 // timestamp
	val int   // data received
}

// data cache used by the algorithm calculating flow from sensor data
//noinspection GoUnusedType
type scratchDataOld struct {
	senData         map[int]sensorData // maps sensors ID with its latest used data
	unusedSampleSum map[int]int        // maps sensors ID with the sum of unused samples received
}

// data cache used by the algorithm calculating flow from sensor data
type scratchData struct {
	senData            map[int]sensorData // maps sensors ID with its latest used data
	unusedSampleSumIn  map[int]int        // maps sensors ID with the sum of unused in samples received
	unusedSampleSumOut map[int]int        // maps sensors ID with the sum of unused out samples received
}
