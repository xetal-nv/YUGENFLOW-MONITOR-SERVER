package gateManager

import "gateserver/dataformats"

// channels to send data from a sensor to the gates it contributes to
var Sensor2Gates map[string]([]chan dataformats.FlowData)
