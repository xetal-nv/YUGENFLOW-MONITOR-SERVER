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
	channelIn  map[string]chan dataformats.SimpleSample
	ChannelOut map[string]chan map[string]dataformats.SimpleSample
}

var RegReferenceChannels struct {
	sync.RWMutex
	channelIn  map[string]chan dataformats.SimpleSample
	ChannelOut map[string]chan map[string]dataformats.SimpleSample
}
