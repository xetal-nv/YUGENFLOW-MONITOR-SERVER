package sensorManager

import "sync"

// basic flow data model used for data from sensors and gates
type SensorDefinition struct {
	Mac        string   `json:"mac"`
	Id         int      `json:"id"`
	Attributes []string `json:"attributes"`
	Active     bool     `json:"active"`
	Disabled   bool     `json:"disabled"`
}

// DeclaredSensors contains the information about a sensor and can be accessed via mac and, indirectly, via ID
// DeclaredSensors is lockable
var DeclaredSensors struct {
	sync.RWMutex
	Mac map[string]SensorDefinition
	Id  map[int]string
}
