package sensorManager

import (
	"gateserver/dataformats"
	"net"
	"sync"
)

// TODO move most to sensorDB

type SensorChannel struct {
	Tcp     net.Conn
	Process chan dataformats.CommandAnswer
}

//ActiveSensors is used as lock and contains the assigned channels
var ActiveSensors struct {
	sync.RWMutex
	Mac map[string]SensorChannel
	Id  map[int]string
}

// type for current sensor configuration
type sensorDefinition struct {
	mac      string
	id       int
	idSent   int
	bypass   bool
	report   bool
	enforce  bool
	strict   bool
	accept   bool
	active   bool
	failures int
	channels SensorChannel
}
