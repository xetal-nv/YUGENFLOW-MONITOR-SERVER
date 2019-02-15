package spaces

import "github.com/pkg/errors"

// sends the gate data to the proper counters
func SendData(gate int, val int) error {
	if gateChannels[gate] == nil {
		return errors.New("Spaces.SendData: error gate not valid")
	}
	for _, v := range gateChannels[gate] {
		go func() { v <- dataGate{num: gate, val: val, group: gateGroup[gate]} }()
	}
	return nil
}
