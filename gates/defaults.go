package gates

import "sync"

const maxDevicePerGate = 2 // maximum number of devices supported per gate

// sensor data as identified id, timestamp ts and value val
type sensorData struct {
	id  int   // sensor number
	ts  int64 // timestamp
	val int   // data received
}

// sensor definition as identified id, is reversed mounted (reversed), to which gates it belongs
// (slice gate) amd to which entry it belongs (slice entry)
type SensorDef struct {
	id       int               // gate or sensor number
	Reversed bool              // reverse flag
	gate     []int             //  gate ids
	entry    []chan sensorData // entry channels for the sensors
}

// entry definition as identified id, list of associated sensors definitions and list of associated gates definitions
type EntryDef struct {
	Id     int               // entry Id
	SenDef map[int]SensorDef // maps sensors ID with its definition
	Gates  map[int][]int     // maps gate Id with its sensor composition
}

// data cache used by the algorithm calculating flow from sensor data
//type scratchDataOld struct {
//	senData         map[int]sensorData // maps sensors ID with its latest used data
//	unusedSampleSum map[int]int        // maps sensors ID with the sum of unused samples received
//}

// data cache used by the algorithm calculating flow from sensor data
type scratchData struct {
	senData            map[int]sensorData // maps sensors ID with its latest used data
	unusedSampleSumIn  map[int]int        // maps sensors ID with the sum of unused in samples received
	unusedSampleSumOut map[int]int        // maps sensors ID with the sum of unused out samples received
}

// Internal variables - some might eb turned into local variables or removed if never used
var sensorList map[int]SensorDef           // all defined sensorList
var gateList map[int][]int                 // list of Gates by device Id, order is preserved from the configuration
var EntryList map[int]EntryDef             // maps devices to entries
var MutexDeclaredDevices = &sync.RWMutex{} // this mutex is used to avoid concurrent DeclaredDevices
var DeclaredDevices map[string]int         // maps the declared mac with the id of a used device
var LogToFileAll bool                      // flag of all entry activity has to be logged to a file
var maximumAsymmetry int                   // maximum difference of activity on sensors per gate, provided via configuration file
var maximumAsymmetryIter int               // maximum times a sensor is restarted for asymmetry before being disabled
var SensorRst struct {                     // array of channels to reset handler
	sync.RWMutex
	Channel map[int]chan bool
}
