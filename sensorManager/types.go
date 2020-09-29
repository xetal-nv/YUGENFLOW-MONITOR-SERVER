package sensorManager

import (
	"gateserver/dataformats"
	"net"
)

type SensorChannel struct {
	tcp         net.Conn
	CmdAnswer   chan dataformats.Commandding
	Commands    chan dataformats.Commandding
	gateChannel []chan dataformats.FlowData
	reset       chan bool
}

// device commands describer for conversion from/to binary to/from param execution
type cmdSpecs struct {
	cmd byte // command binary value
	lgt int  // number of bytes of arguments excluding cmd (1 byte) and the id (2 bytes)
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

// sensorSpecs captures the data for setSensorParameters
type sensorSpecs struct {
	srate int
	savg  int
	bgth  float64
	occth float64
}
