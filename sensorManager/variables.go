package sensorManager

import (
	"sync"
	"time"
)

// maximum number of TCP connections
const MAXTCP = 200

// how many times the server tries to reset the sensor eeprom before reporting an error
const eepromResetTries = 3

// how many times the server tries to execute the enforce tag before disconnecting the sensor
const enforceTries = 10

// contains sensor EERPROM values (if applicable)
var commonSensorSpecs sensorSpecs
var sensorData map[string]sensorSpecs

// this channel is used tor regulate the number of active sensors
var tokens chan interface{}

//ActiveSensors is used as lock and contains the assigned channels
var ActiveSensors struct {
	sync.RWMutex
	Mac map[string]SensorChannel
	Id  map[int]string
}

// provides length for legal server2gate commands
var cmdAnswerLen = map[byte]int{
	2:  1,
	3:  1,
	4:  1,
	5:  1,
	6:  1,
	8:  1,
	10: 1,
	12: 1,
	13: 1,
	14: 1,
	7:  3,
	9:  3,
	11: 3,
}

var resetBG struct {
	start    time.Time
	end      time.Time
	interval time.Duration
	valid    bool
}

// provides length for legal server2gate commands
// server also has commands
// list : lists all commands
// macid " assigns the id at the device with mac specified in val
// lgt is max 4 (bytes)
var cmdAPI = map[string]cmdSpecs{
	"srate":     {2, 1},
	"savg":      {3, 1},
	"bgth":      {4, 2},
	"occth":     {5, 2},
	"rstbg":     {6, 0},
	"readdiff":  {7, 0},
	"resetdiff": {8, 0},
	"readinc":   {9, 0},
	"rstinc":    {10, 0},
	"readoutc":  {11, 0},
	"rstoutc":   {12, 0},
	"readid":    {13, 0},
	"setid":     {14, 2},
}
