package spaces

import (
	"sync"
	"time"
)

// defines an averaging interval
type avgInterval struct {
	name     string
	interval int
}

// define the interval when every counter needs to be forced to zero
type TimeSchedule struct {
	Start    time.Time
	End      time.Time
	Duration int64
}

// define intervals and stores values for presence detectors
type IntervalDetector struct {
	Id         string    // entry Id as string to support entry data in the entire communication pipe
	Start, End time.Time // Start and End of the interval
	incycle    bool      // track if it si in cycle
	Activity   DataEntry // Activity count
}

type pfunc func(string, spaceEntries) interface{}
type cfunc func(string, chan interface{}, chan bool)
type dtfuncs struct {
	pf pfunc
	cf cfunc
}

// Constants
const chantimeout = 100
const minTransactionsForDetection = 2

// variables defined via options/configuration file
var Crashmaxdelay int64
var multicycleonlydays bool

// Internal variables - some might be turned into local variables
var dtypes map[string]dtfuncs                                      // holds the datatypes and the associated prep functions for space.passData
var instNegSkip bool                                               // skips instantaneous negative counters
var avgNegSkip bool                                                // skips instantaneous negative counters
var bufsize int                                                    // size of channel buffer among samplers
var entrySpaceSamplerChannels map[int][]chan spaceEntries          // channels from entry to associated space sampler
var entrySpacePresenceChannels map[int][]chan spaceEntries         // channels from entry to associated space presence detector
var SamplingWindow int                                             // internal for the averaging of data
var avgAnalysis []avgInterval                                      // specification sampling data for visualisation
var avgAnalysisSchedule TimeSchedule                               // specifies the Activity range of the analysis
var latestChannelLock = &sync.RWMutex{}                            // this mutex is for a perceived race on the below slices
var latestBankIn map[string]map[string]map[string]chan interface{} // contains all input channels to the data bank
var latestDBSIn map[string]map[string]map[string]chan interface{}  // contains all input channels to the database

// external variables
var _ResetDBS map[string]map[string]map[string]chan bool            // reset channel for the DBS's
var LatestBankOut map[string]map[string]map[string]chan interface{} // contains all output channels to the data bank
var LatestDetectorOut map[string]chan []IntervalDetector            // contains the latest presence values for recovery purposes
var SpaceDef map[string][]int                                       // maps a space name to its entries
var SpaceMaxOccupancy map[string]int                                // maps a space name to its maximum occupancy, if defined
var SpaceTimes map[string]TimeSchedule                              // maps a space name to its closure times
var cmode string                                                    // data compression mode
var MutexInitData = &sync.RWMutex{}                                 // mutex for InitData
var InitData map[string]map[string]map[string][]string              // holds values from a previous run loaded from file .recoveryavg
