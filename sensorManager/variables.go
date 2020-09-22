package sensorManager

import (
	"net"
	"sync"
)

// TODO cmove most to sensorDB

//ActiveSensors is used as lock and contains the assigned channels
var ActiveSensors struct {
	sync.RWMutex
	Mac map[string]SensorDefinition
	Id  map[int]string
}

// basic flow data model used for data from sensors and gates
type SensorDefinition struct {
	Mac                 string   `json:"mac"`
	Id                  int      `json:"id"`
	Bypass              bool     `json:"bypass"`
	Report              bool     `json:"report"`
	Enforce             bool     `json:"enforce"`
	Strict              bool     `json:"strict"`
	CurrentChannel      net.Conn `json:-`
	SuspectedConnection int      `json:-`
}

// Sensors is lockable
var DeclaredSensors struct {
	sync.RWMutex
	Mac map[string]SensorDefinition
	Id  map[int]string
}

// MaliciousDevices stored all suspected devices (map[mac]ip) classified in suspected and disabled
var MaliciousDevices struct {
	sync.RWMutex `json:"-"`
	Suspected    map[string]string `json:"suspectedDevices"`
	Disabled     map[string]string `json:"disabledDevices"`
}

// MaliciousIPS stores all suspected IP's, the boolean indicated if it is suspected (false) or disabled (true)
var MaliciousIPS struct {
	sync.RWMutex `json:"-"`
	Disabled     map[string]bool `json:"disabledIPS"`
}
