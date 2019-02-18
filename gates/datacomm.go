package gates

import (
	"countingserver/support"
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
				val = 255 - val
			}
		}
		for _, c := range v.entry {
			go func() { c <- sensorData{num: dev, val: val, ts: support.Timestamp()} }()
		}
		return nil
	} else {
		return errors.New("gates.SendData: error device not valid, ID: " + strconv.Itoa(dev))
	}
}
