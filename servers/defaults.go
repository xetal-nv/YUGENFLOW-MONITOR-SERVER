package servers

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

const SIZE int = 2

// device commands describer for conversion from/to binary to/from param execution
type cmdspecs struct {
	cmd byte // command binary value
	lgt int  // nuber of bytes of arguments exclusing cmd (1 byte) and the id (2 bytes)
}

type datafunc func() GenericData

var addServer [SIZE]string                  // server addresses
var sdServer [SIZE + 1]chan context.Context // channel for closure of servers
var hMap [SIZE]map[string]http.Handler      // server handler maps
var crcUsed bool                            // CRC used flag
var strictFlag bool                         // indicate is MAC strict mode is being used
//var cmdBuffLen int                          // length of buffer for command channels
var mutexSensorMacs = &sync.RWMutex{}       // this mutex is used to avoid concurrent writes on start-up on sensorMacID, sensorMacID,SensorCmdID
var mutexUnknownMac = &sync.RWMutex{}       // this mutex is used to avoid concurrent writes on unknownMacChan
var mutexUnusedDevices = &sync.RWMutex{}    // this mutex is used to avoid concurrent writes at start-up on sensorMacID, sensorMacID,SensorCmdID
var unknownMacChan map[string]chan net.Conn // map of sensor id to sensor MAC as provided by the sensor itself
var unkownDevice map[string]bool            // map of mac of devices registered with id equal to 0xff
var unusedDevice map[int]string             // map of id's of unused registered deviced (as in not in the .env file)
var sensorMacID map[int][]byte              // map of sensor id to sensor MAC as provided by the sensor itself
var sensorIdMAC map[string]int              // map of sensor MAC to sensor id as provided by the sensor itself
var sensorChanID map[int]chan []byte        // channel to handler managing commands to each connected sensor
var sensorChanUsedID map[int]bool           // flag indicating if thw channel is assigned to a TCP handler
var SensorCmdID map[int]chan []byte         // externally visible channel to handler managing commands to each connected sensor
var dataMap map[string]datafunc             // used for HTTP API handling of different data types
var cmdAnswerLen = map[byte]int{            // provides length for legal server2gate commands
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
var timeout, maltimeout int
var resetbg struct {
	start    time.Time
	end      time.Time
	interval time.Duration
	valid    bool
}

// index for the map[string]string argument of exeParamCommand
var cmds = []string{"cmd", "val", "async", "id", "timeout"}

// provides length for legal server2gate commands
var cmdAPI = map[string]cmdspecs{
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

const maxsensors = 150               // maximum number of allowed processors
const mindelayrefusedconnection = 30 // mininum delay for refused connection
var tcpTokens chan bool              // token for accepting a TCP erquest
