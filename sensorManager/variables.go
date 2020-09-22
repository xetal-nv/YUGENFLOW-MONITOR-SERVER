package sensorManager

import (
	"gateserver/dataformats"
	"net"
	"sync"
)

// TODO move most to sensorDB

type SensorChannel struct {
	Tcp     net.Conn
	Process chan dataformats.SensorCommand
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
	bypass   bool
	report   bool
	enforce  bool
	strict   bool
	channels SensorChannel
}

// TODO var below to be moved to the database
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
