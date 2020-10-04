package spaceManager

import (
	"gateserver/dataformats"
	"sync"
)

// channels to send data from an entry to the spaces it contributes to
var EntryStructure struct {
	sync.RWMutex
	SpaceList   map[string][]string
	DataChannel map[string]([]chan dataformats.Entrydata)
}

// channels to connect to a space
var SpaceStructure struct {
	sync.RWMutex
	EntryList   map[string]map[string]dataformats.GateDefinition
	DataChannel map[string]chan dataformats.Entrydata
	SetReset    map[string]chan bool
	StopChannel map[string]chan interface{}
}
