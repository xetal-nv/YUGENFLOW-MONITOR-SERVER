package spaces

import "countingserver/registers"

type dataChan struct {
	num   int // gate number
	val   int // data received
	group int // group id
}

type avgInterval struct {
	name     string
	interval int
}

// Internal variables
var negSkip bool                           // skips instantaneous negative counters
var spaceChannels map[string]chan dataChan // maps space to its associated data channel
var gateChannels map[int][]chan dataChan   // maps gate to the channels/spaces it belongs to
var gateGroup map[int]int                  // maps gate to group_id
var reversedGates []int                    // list of gates with reversed counters
var groupsStats map[int]int                // gives size og group per group_id
var samplingWindow int                     // internal for the averaging of data
var avgAnalysis []avgInterval              // specification sampling data for visualisation

// external variables

var LatestDataBankOut map[string]map[string]chan registers.DataCt
var LatestDataBankIn map[string]map[string]chan registers.DataCt
