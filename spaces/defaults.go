package spaces

import "countingserver/registers"

type dataGate struct {
	num   int   // gate number
	val   int   // data received
	group int   // group id
	ts    int64 // timestamp
}

type avgInterval struct {
	name     string
	interval int
}

// Internal variables
var instNegSkip bool                                             // skips instantaneous negative counters
var avgNegSkip bool                                              // skips instantaneous negative counters
var bufsize int                                                  // size of channel buffer among samplers
var gateChannels map[int][]chan dataGate                         // maps gate to the channels/spaces it belongs to
var gateGroup map[int]int                                        // maps gate to group_id
var reversedGates []int                                          // list of gates with reversed counters
var groupsStats map[int]int                                      // gives size og group per group_id
var samplingWindow int                                           // internal for the averaging of data
var avgAnalysis []avgInterval                                    // specification sampling data for visualisation
var latestDataBankIn map[string]map[string]chan registers.DataCt // input channels to registry
var latestDataDBSIn map[string]map[string]chan registers.DataCt  // input channels to databases

// external variables
var LatestDataBankOut map[string]map[string]chan registers.DataCt // output channels to registry
var latestDataDBSOut map[string]map[string]chan registers.DataCt  // input channels to databases
