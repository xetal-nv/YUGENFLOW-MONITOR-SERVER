package sensorManager

import (
	"net"
	"sync"
)

// basic flow data model used for data from sensors and gates
type SensorDefinition struct {
	Mac string `json:"mac"`
	Id  int    `json:"id"`
	//Attributes []string `json:"attributes"`
	Bypass              bool     `json:"bypass"`
	Report              bool     `json:"bypreportass"`
	Enforce             bool     `json:"enforce"`
	Strict              bool     `json:"strict"`
	Active              bool     `json:"active"`
	Disabled            bool     `json:"disabled"`
	CurrentChannel      net.Conn `json:"-"`
	SuspectedConnection int      `json:"-"`
}

// DeclaredSensors contains the information about a sensor and can be accessed via mac and, indirectly, via ID
// DeclaredSensors is lockable
var DeclaredSensors struct {
	sync.RWMutex
	Mac map[string]SensorDefinition
	Id  map[int]string
}
