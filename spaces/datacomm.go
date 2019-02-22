package spaces

import (
	"github.com/pkg/errors"
	"strconv"
)

// sends the gate data to the proper counters
func SendData(entry int, val int) error {
	if entrySpaceChannels[entry] == nil {
		return errors.New("spaces.SendData: error entry not valid id: " + strconv.Itoa(entry))
	}
	for _, v := range entrySpaceChannels[entry] {
		go func() { v <- spaceEntries{val: val} }()
	}
	return nil
}
