package spaces

type avgInterval struct {
	name     string
	interval int
}

// Constants
const chantimeout = 100

// Internal variables - some might be turned into local variables
var instNegSkip bool                                               // skips instantaneous negative counters
var avgNegSkip bool                                                // skips instantaneous negative counters
var bufsize int                                                    // size of channel buffer among samplers
var entrySpaceChannels map[int][]chan spaceEntries                 // channels form entry to associated space
var samplingWindow int                                             // internal for the averaging of data
var avgAnalysis []avgInterval                                      // specification sampling data for visualisation
var latestBankIn map[string]map[string]map[string]chan interface{} // contains all input channels to the data bank
var latestDBSIn map[string]map[string]map[string]chan interface{}  // contains all input channels to the database

//var latestDataBankIn map[string]map[string]chan interface{}  // input channels to registry for samples
//var latestEntryBankIn map[string]map[string]chan interface{} // input channels to registry for entries
//var latestDataDBSIn map[string]map[string]chan interface{}   // input channels to databases for samples
//var latestEntryDBSIn map[string]map[string]chan interface{} // input channels to databases for entry

// external variables
var ResetDBS map[string]map[string]map[string]chan bool             // reset channel for the DBS's
var LatestBankOut map[string]map[string]map[string]chan interface{} // contains all input channels to the data bank

//var LatestDataBankOut map[string]map[string]chan interface{}  // output channels to registry for samples
//var LatestEntryBankOut map[string]map[string]chan interface{} // output channels to registry for entries
//var ResetDataDBS map[string]map[string]chan bool              // reset channel for a given Data DBS for samples
//var ResetEntryDBS map[string]map[string]chan bool             // reset channel for a given Data DBS for entries
