// +build newcache

package diskCache

import (
	"encoding/json"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
)

func ReadState(spaceName string) (space dataformats.SpaceState, err error) {
	if msg, found := recovery.Get(spaceName); found {
		if err = json.Unmarshal([]byte(msg), &space); err != nil {
			return space, globals.SensorDBError
		}
	}
	return
}

func SaveState(space dataformats.SpaceState) (err error) {
	bdr, err := json.Marshal(space)
	if err != nil {
		return globals.SensorDBError
	}
	recovery.Set(space.Id, string(bdr), jac.DefaultExpiration, true)
	return
}

func DeleteState(spaceName string) (err error) {
	recovery.Delete(spaceName)
	return
}

func ReadShadowState(spaceName string) (space dataformats.SpaceState, err error) {
	if msg, found := shadowRecovery.Get(spaceName); found {
		if err = json.Unmarshal([]byte(msg), &space); err != nil {
			return space, globals.SensorDBError
		}
	}
	return
}

func SaveShadowState(space dataformats.SpaceState) (err error) {
	bdr, err := json.Marshal(space)
	if err != nil {
		return globals.SensorDBError
	}
	shadowRecovery.Set(space.Id, string(bdr), jac.DefaultExpiration, true)
	return
}

func DeleteShadowState(spaceName string) (err error) {
	shadowRecovery.Delete(spaceName)
	return
}
