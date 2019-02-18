package gates

type sensorData struct {
	num int   // gate number
	val int   // data received
	ts  int64 // timestamp
}

type sensorDef struct {
	id       int             // gate number
	reversed bool            // reverse flag
	gate     int             //  gate id
	entry    chan sensorData // entry id
}

// Internal variables - some might eb turned into local variables or removed if never used
var sensorList map[int]sensorDef // all defined sensorList
var gateList map[int][]int       // list of gates by device composinio and id
var entryList map[int][]int      // list of entries by device composinio and id
