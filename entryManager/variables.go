package entryManager

import (
	"gateserver/dataformats"
	"sync"
)

var saveToDB bool

// channels to send data from a gate to the entries it contributes to
var GateStructure struct {
	sync.RWMutex
	EntryList   map[string][]string
	DataChannel map[string]([]chan dataformats.FlowData)
}

// channels to connect to an entry
var EntryStructure struct {
	sync.RWMutex
	GateList    map[string]map[string]dataformats.GateState
	DataChannel map[string]chan dataformats.FlowData
	SetReset    map[string]chan bool
	StopChannel map[string]chan interface{}
}
