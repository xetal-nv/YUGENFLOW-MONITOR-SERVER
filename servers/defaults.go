package servers

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const SIZE int = 2

type datafunc func() GenericData

var addServer [SIZE]string                  // server addresses
var sdServer [SIZE + 1]chan context.Context // channel for closure of servers
var hMap [SIZE]map[string]http.Handler      // server handler maps
var crcUsed bool                            // CRC used flag
//var cmdBuffLen int                          // length of buffer for command channels
var mutexSensorMaos = &sync.Mutex{} // this mutex is used to avoid concurrent writes at start-up on sensorMac, sensorMac,SensorCmd
var sensorMac map[int][]byte        // map of sensor id to sensor MAC as provided by the sensor itself
var sensorChan map[int]chan []byte  // channel to handler managing commands to each connected sensor
var SensorCmd map[int]chan []byte   // externally visible channel to handler managing commands to each connected sensor
var dataMap map[string]datafunc     // used for HTTP API handling of different data types
var cmdAnswerLen = map[byte]int{    // provides length for legal server2gate commands
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
var timeout int
var resetbg struct {
	start    time.Time
	end      time.Time
	interval time.Duration
	valid    bool
}
