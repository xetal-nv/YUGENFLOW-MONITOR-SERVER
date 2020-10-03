package gateManager

import (
	"gateserver/dataformats"
	"sync"
)

var saveToDB bool

// channels to send data from a sensor to the gates it contributes to
var SensorStructure struct {
	sync.RWMutex
	GateList    map[int][]string
	DataChannel map[int]([]chan dataformats.FlowData)
}

// channels to connect to a gates
var GateStructure struct {
	sync.RWMutex
	SensorList         map[string]map[int]dataformats.SensorDefinition
	DataChannel        map[string]chan dataformats.FlowData
	ConfigurationReset map[string]chan interface{}
	StopChannel        map[string]chan interface{}
}
