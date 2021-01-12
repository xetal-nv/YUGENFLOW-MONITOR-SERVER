// +build newcache

package diskCache

import (
	"encoding/json"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
)

func ReadSnapshot(space string) (dr dataformats.MeasurementSampleWithFlows, err error) {
	if msg, found := recovery.Get(space); found {
		if err = json.Unmarshal([]byte(msg), &dr); err != nil {
			return dr, globals.SensorDBError
		}
	}
	return
}

func DeleteSnapshot(space string) (err error) {
	recovery.Delete(space)
	return
}

func SaveSnapshot(dr dataformats.MeasurementSampleWithFlows) (err error) {
	bdr, err := json.Marshal(dr)
	if err != nil {
		return globals.SensorDBError
	}
	recovery.Set(dr.Space, string(bdr), jac.DefaultExpiration, true)
	return
}
