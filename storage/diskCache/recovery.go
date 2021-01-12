// +build !newcache

package diskCache

import (
	"encoding/json"
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
)

func ReadSnapshot(space string) (dr dataformats.MeasurementSampleWithFlows, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		err = json.Unmarshal(tx.Bucket([]byte(recovery)).Get([]byte(space)), &dr)
		return err
	})
	return
}

func SaveSnapshot(dr dataformats.MeasurementSampleWithFlows) error {
	bdr, err := json.Marshal(dr)
	if err != nil {
		return err
	}
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(recovery)).Put([]byte(dr.Space), bdr)
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

func DeleteSnapshot(space string) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(recovery)).Delete([]byte(space))
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
