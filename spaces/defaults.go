package spaces

type avgInterval struct {
	name     string
	interval int
}

type pfunc func(string, spaceEntries) interface{}
type cfunc func(string, chan interface{}, chan bool)
type dtfuncs struct {
	pf pfunc
	cf cfunc
}

// Constants
const chantimeout = 100

// Internal variables - some might be turned into local variables
var dtypes map[string]dtfuncs                                      // holds the datatypes and the associated prep functions for space.passData
var instNegSkip bool                                               // skips instantaneous negative counters
var avgNegSkip bool                                                // skips instantaneous negative counters
var bufsize int                                                    // size of channel buffer among samplers
var entrySpaceChannels map[int][]chan spaceEntries                 // channels form entry to associated space
var samplingWindow int                                             // internal for the averaging of data
var avgAnalysis []avgInterval                                      // specification sampling data for visualisation
var latestBankIn map[string]map[string]map[string]chan interface{} // contains all input channels to the data bank
var latestDBSIn map[string]map[string]map[string]chan interface{}  // contains all input channels to the database

// external variables
var ResetDBS map[string]map[string]map[string]chan bool             // reset channel for the DBS's
var LatestBankOut map[string]map[string]map[string]chan interface{} // contains all input channels to the data bank
