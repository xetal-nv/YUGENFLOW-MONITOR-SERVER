// +build !newcache

package diskCache

import (
	"encoding/json"
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
)

func ReadState(spaceName string) (space dataformats.SpaceState, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(recovery)).Get([]byte(spaceName))
		if data == nil {
			return globals.KeyInvalid
		}
		return json.Unmarshal(data, &space)
	})
	return
}

func SaveState(space dataformats.SpaceState) error {
	data, err := json.Marshal(space)
	if err != nil {
		return err
	}
	err = main.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(recovery)).Put([]byte(space.Id), data)
		if err != nil {
			if globals.DebugActive {
				fmt.Println(err.Error())
			}
			return globals.SensorDBError
		}
		return nil
	})
	return err
}

func DeleteState(spaceName string) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(recovery)).Delete([]byte(spaceName))
		if err != nil {
			if globals.DebugActive {
				fmt.Println(err.Error())
			}
			return globals.SensorDBError
		}
		return nil
	})
	return
}

func ReadShadowState(spaceName string) (space dataformats.SpaceState, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(shadowRecovery)).Get([]byte(spaceName))
		if data == nil {
			return globals.KeyInvalid
		}
		return json.Unmarshal(data, &space)
	})
	return
}

func SaveShadowState(space dataformats.SpaceState) error {
	data, err := json.Marshal(space)
	if err != nil {
		return err
	}
	err = main.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(shadowRecovery)).Put([]byte(space.Id), data)
		if err != nil {
			if globals.DebugActive {
				fmt.Println(err.Error())
			}
			return globals.SensorDBError
		}
		return nil
	})
	return err
}

func DeleteShadowState(spaceName string) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(shadowRecovery)).Delete([]byte(spaceName))
		if err != nil {
			if globals.DebugActive {
				fmt.Println(err.Error())
			}
			return globals.SensorDBError
		}
		return nil
	})
	return
}
