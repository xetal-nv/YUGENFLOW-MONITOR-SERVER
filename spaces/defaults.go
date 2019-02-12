package spaces

type dataChan struct {
	num   int
	val   int
	group int
}

var spaceChannels map[string]chan dataChan // maps space to its associated data channel
var gateChannels map[int][]chan dataChan   // maps gate to the channels/spaces it belongs to
var gateGroup map[int]int                  // maps gate to group_id
var reversedGates []int                    // list of gates with reversed counters

var GroupsStats map[int]int // gives size og group per group_id
