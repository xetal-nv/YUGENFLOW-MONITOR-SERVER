package spaces

type avgInterval struct {
	name     string
	interval int
}

// Internal variables - some might be turned into local variables
var instNegSkip bool                                        // skips instantaneous negative counters
var avgNegSkip bool                                         // skips instantaneous negative counters
var bufsize int                                             // size of channel buffer among samplers
var entrySpaceChannels map[int][]chan spaceEntries          // channels form entry to associated space
var samplingWindow int                                      // internal for the averaging of data
var avgAnalysis []avgInterval                               // specification sampling data for visualisation
var latestDataBankIn map[string]map[string]chan interface{} // input channels to registry
var latestDataDBSIn map[string]map[string]chan interface{}  // input channels to databases

// external variables
var LatestDataBankOut map[string]map[string]chan interface{} // output channels to registry
var ResetDataDBS map[string]map[string]chan bool             // reset channel for a given Data DBS
