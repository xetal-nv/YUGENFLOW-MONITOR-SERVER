package gates

type sensorData struct {
	id  int   // gate number
	ts  int64 // timestamp
	val int   // data received
}

type SensorDef struct {
	id       int               // gate number
	Reversed bool              // reverse flag
	gate     []int             //  gate ids
	entry    []chan sensorData // entry channels for the sensors
}

type EntryDef struct {
	Id     int               // entry Id
	SenDef map[int]SensorDef // maps sensors ID with its definition
	Gates  map[int][]int     // maps gate Id with its sensor composition
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
var sensorList map[int]SensorDef // all defined sensorList
var gateList map[int][]int       // list of Gates by device Id, order is preserved from the configuration
var EntryList map[int]EntryDef   // maps devices to entries
