package gates

import (
	"countingserver/support"
	"fmt"
	"github.com/pkg/errors"
	"strconv"
)

// sends the gate data to the proper counters
func SendData(dev int, val int) error {
	if v, ok := sensorList[dev]; ok {

		if v.entry == nil {
			return errors.New("gates.SendData: error device not valid, ID: " + strconv.Itoa(dev))
		}
		if v.reversed {
			if val != 127 {
				val = 256 - val
			}
		}
		for _, c := range v.entry {
			// convert to int from int8 with 127 as special value
			if val == 127 {
				val = 0
				//support.DLog <- support.DevData{"device " + strconv.Itoa(dev), support.Timestamp(), "127 reported", nil}
			} else {
				val = int(int8(val & 255))
			}
			if support.Debug > 0 {
				fmt.Printf("\nDevice %v sent data %v at %v\n", dev, val, support.Timestamp())
			}
			go func() { c <- sensorData{num: dev, val: val, ts: support.Timestamp()} }()
		}
		return nil
	} else {
		return errors.New("gates.SendData: error device not valid, ID: " + strconv.Itoa(dev))
	}
}
