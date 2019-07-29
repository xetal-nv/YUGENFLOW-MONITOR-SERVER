package spaces

import (
	"github.com/pkg/errors"
	"strconv"
)

// Implement entry data distribution
func SendData(entry int, val int) error {
	// This function takes care of distributing de received data to the relevant processors

	// sends the gate data to the proper counters
	if entrySpaceSamplerChannels[entry] == nil {
		return errors.New("spaces.SendData: error entry not valid Id: " + strconv.Itoa(entry))
	}
	for _, v := range entrySpaceSamplerChannels[entry] {
		go func() { v <- spaceEntries{id: entry, val: val} }()
	}

	//  sends the gate data to the proper presence detectors
	if entrySpacePresenceChannels[entry] == nil {
		return errors.New("spaces.SendData: error entry not valid Id: " + strconv.Itoa(entry))
	}
	for _, v := range entrySpacePresenceChannels[entry] {
		go func() { v <- spaceEntries{id: entry, val: val} }()
	}

	return nil
}
