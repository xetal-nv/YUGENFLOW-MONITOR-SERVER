package sensorManager

import (
	"gateserver/dataformats"
	"net"
)

type SensorChannel struct {
	tcp         net.Conn
	CmdAnswer   chan dataformats.Commanding
	Commands    chan dataformats.Commanding
	gateChannel []chan dataformats.FlowData
	reset       chan bool
}

// device commands describer for conversion from/to binary to/from param execution
type CmdSpecs struct {
	Cmd byte // command binary value
	Lgt int  // number of bytes of arguments excluding Cmd (1 byte) and the id (2 bytes)
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
