package servers

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

const maxSensors = 200                      // maximum number of allowed processors
const minDelayRefusedConnection = 30        // minimum delay for refused connection
const sensorEEPROMfile = "sensors.settings" // file containing the sensor eerpom values
const eepromResetTries = 3                  // how many times the server tries to reset the sensor eeprom before reporting an error

// device commands describer for conversion from/to binary to/from param execution
type cmdSpecs struct {
	cmd byte // command binary value
	lgt int  // number of bytes of arguments excluding cmd (1 byte) and the id (2 bytes)
}

// sensorSpecs captures the data for setSensorParameters

type sensorSpecs struct {
	srate int
	savg  int
	bgth  float64
	occth float64
}

type dataFunc func() GenericData

//var addServer [SIZE]string                  // server address
var addServer string                 // server addresses
var sdServer [2]chan context.Context // channel for closure of servers
//var hMap [SIZE]map[string]http.Handler      // server handler maps
var hMap map[string]http.Handler            // server handler maps
var crcUsed bool                            // CRC used flag
var TCPdeadline int                         // TCP read deadline in hours (default is 24)
var strictFlag bool                         // indicate is MAC strict mode is being used
var mutexSensorMacs = &sync.RWMutex{}       // this mutex is used to avoid concurrent writes on start-up on sensorMacID, sensorMacID, SensorCmdID, SensorCmdMac
var mutexUnknownMac = &sync.RWMutex{}       // this mutex is used to avoid concurrent writes on unknownMacChan
var mutexPendingDevices = &sync.RWMutex{}   // this mutex is used to avoid concurrent writes on mutexPendingDevices
var mutexUnusedDevices = &sync.RWMutex{}    // this mutex is used to avoid concurrent writes on unusedDevice
var mutexConnMAC = &sync.RWMutex{}          // this mutex is used to avoid concurrent writes on sensorConnMAC
var unknownMacChan map[string]chan net.Conn // map of sensor id to sensor MAC as provided by the sensor itself
var pendingDevice map[string]bool           // map of mac of devices pending registration
var unknownDevice map[string]bool           // map of mac of devices registered with id equal to 0xff
var unusedDevice map[int]string             // map of id's of unused registered devices (as in not in the .env file)
var sensorMacID map[int][]byte              // map of sensor id to sensor MAC as provided by the sensor itself
var sensorIdMAC map[string]int              // map of sensor MAC to sensor id as provided by the sensor itself
var sensorConnMAC map[string]net.Conn       // map of sensor MAC to the tcp channel
var sensorChanID map[int]chan []byte        // channel to handler managing commands to each connected sensor
var sensorChanUsedID map[int]bool           // flag indicating if thw channel is assigned to a TCP handler
var SensorCmdID map[int]chan []byte         // externally visible channel to handler managing commands to each connected sensor via ID
var SensorCmdMac map[string][]chan []byte   // externally visible channel to handler managing commands to each connected sensor via mac
var SensorIDCMDMac map[string]chan int      // externally visible channel to handler managing commands to each connected sensor via mac
var dataMap map[string]dataFunc             // used for HTTP API handling of different data types
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
var timeout, malTimeout int
var resetBG struct {
	start    time.Time
	end      time.Time
	interval time.Duration
	valid    bool
}

// index for the map[string]string argument of exeParamCommand
var commandNames = []string{"cmd", "val", "async", "id", "timeout", "mac"}

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

var errorMngt = [3]int{1, 5, 15}      // [min penalty, max penalty, max number of consecutive errors]
var tcpTokens chan bool               // token for accepting a TCP request
var KSwitch bool                      // kill switch flag
var RepCon bool                       // enables reporting on current
var commonSensorSpecs sensorSpecs     // specs valid for all sensors
var sensorData map[string]sensorSpecs // specs valid for a given sensor mac
var SensorEEPROMResetEnabled bool     // if true the sensor EEPROM is reset at every connection
var sensorEEPROMResetDelay int        // number of seconds of delay before initiating the eeprom refresh
var sensorEEPROMResetStep int         // number of seconds of delay between refresh commands
var EnableDBSApi bool                 // enables the DBS update from file API

// debug access control
var dbgMutex = &sync.RWMutex{}   // lock to dbgRegistry
var dbgRegistry map[string]int64 // registry of currently authorised IPs
const authDbgInterval = 60       // authorisation interval for debug access in minutes
const pinDbg = "pippopluto"      // debug pin
