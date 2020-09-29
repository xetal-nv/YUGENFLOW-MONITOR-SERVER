package gateManager

import (
	"gateserver/dataformats"
	"sync"
)

// channels to send data from a sensor to the gates it contributes to
var SensorList struct {
	sync.RWMutex
	GateList    map[int][]string
	DataChannel map[int]([]chan dataformats.FlowData)
}

// channels to connect to a gates
var GateList struct {
	sync.RWMutex
	SensorList  map[string]map[int]dataformats.SensorDefinition
	DataChannel map[string]chan dataformats.FlowData
	StopChannel map[string]chan interface{}
}
