package avgsManager

import (
	"gateserver/dataformats"
	"sync"
)

var LatestData struct {
	sync.RWMutex
	Channel map[string]chan dataformats.SpaceState
}

var RegRealTimeChannels struct {
	sync.RWMutex
	channelIn  map[string]chan dataformats.MeasurementSample
	ChannelOut map[string]chan map[string]dataformats.MeasurementSample
}

var RegReferenceChannels struct {
	sync.RWMutex
	channelIn  map[string]chan dataformats.MeasurementSample
	ChannelOut map[string]chan map[string]dataformats.MeasurementSample
}

var RegActualChannels struct {
	sync.RWMutex
	channelIn  map[string]chan dataformats.MeasurementSampleWithFlows
	ChannelOut map[string]chan dataformats.MeasurementSampleWithFlows
}
