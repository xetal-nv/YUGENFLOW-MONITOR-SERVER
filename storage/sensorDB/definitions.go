package sensorDB

import (
	"encoding/json"
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
)

func ReadDefinition(mac []byte) (dr dataformats.SensorDefinition, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		err = json.Unmarshal(tx.Bucket([]byte(definitions)).Get([]byte(mac)), &dr)
		return err
	})
	return
}

func WriteDefinition(mac []byte, dr dataformats.SensorDefinition) error {
	bdr, err := json.Marshal(dr)
	if err != nil {
		return err
	}
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(definitions)).Put([]byte(mac), bdr)
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

func DeleteDefinition(mac []byte) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(definitions)).Delete([]byte(mac))
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
