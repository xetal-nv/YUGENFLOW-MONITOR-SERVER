package sensorDB

import (
	"fmt"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
)

func LookUpMac(id []byte) (mac string, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		macb := tx.Bucket([]byte(lookup)).Get(id)
		if macb == nil {
			return globals.KeyInvalid
		}
		mac = string(macb)
		return nil
	})
	return
}

func AddLookUp(id []byte, mac string) error {
	macb := []byte(mac)
	err := main.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(lookup)).Put(id, macb)
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

func DeleteLookUp(id []byte) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(lookup)).Delete(id)
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
