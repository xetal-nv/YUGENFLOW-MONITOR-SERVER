package spaces

import "github.com/pkg/errors"

// sends the gate data to the proper counters
func SendData(gate int, val int) error {
	if entrySpaceChannels[gate] == nil {
		return errors.New("Spaces.SendData: error gate not valid")
	}
	for _, v := range entrySpaceChannels[gate] {
		go func() { v <- dataEntry{num: gate, val: val} }()
	}
	return nil
}
