package avgsManager

import (
	"gateserver/dataformats"
	"sync"
)

var LatestData struct {
	sync.RWMutex
	Channel map[string]chan dataformats.SpaceState
}

var RegisterChannels struct {
	sync.RWMutex
	channelIn  map[string]chan dataformats.SimpleSample
	ChannelOut map[string]chan dataformats.SimpleSample
}
