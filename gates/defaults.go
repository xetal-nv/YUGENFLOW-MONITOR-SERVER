package gates

type sensorData struct {
	id  int   // gate number
	ts  int64 // timestamp
	val int   // data received
}

type sensorDef struct {
	id       int               // gate number
	reversed bool              // reverse flag
	gate     []int             //  gate ids
	entry    []chan sensorData // entry channels for the sensors
}

type entryDef struct {
	id     int               // entry id
	senDef map[int]sensorDef // maps sensors ID with its definition
	gates  map[int][]int     // maps gate id with its sensor composition
}

type scratchDataOld struct {
	senData         map[int]sensorData // maps sensors ID with its latest used data
	unusedSampleSum map[int]int        // maps sensors ID with the sum of unused samples received
}

type scratchData struct {
	senData            map[int]sensorData // maps sensors ID with its latest used data
	unusedSampleSumIn  map[int]int        // maps sensors ID with the sum of unused in samples received
	unusedSampleSumOut map[int]int        // maps sensors ID with the sum of unused out samples received
}

// Internal variables - some might eb turned into local variables or removed if never used
var sensorList map[int]sensorDef // all defined sensorList
var gateList map[int][]int       // list of gates by device id, order is preserved from the configuration
var EntryList map[int]entryDef   // maps devices to entries
