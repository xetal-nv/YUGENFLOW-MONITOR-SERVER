package gates

type sensorData struct {
	num int   // gate number
	val int   // data received
	ts  int64 // timestamp
}

type sensorDef struct {
	id       int               // gate number
	reversed bool              // reverse flag
	gate     []int             //  gate ids
	entry    []chan sensorData // entry channels for the sensors
}

// Internal variables - some might eb turned into local variables or removed if never used
var sensorList map[int]sensorDef // all defined sensorList
var gateList map[int][]int       // list of gates by device id, order is preserved from the configuration
var entryList map[int][]int      // list of entries by device composinio and id
