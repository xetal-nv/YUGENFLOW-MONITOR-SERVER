package sensorDB

import (
	"encoding/binary"
	"fmt"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
)

func CheckMAC(mac []byte, threshold int) (danger bool, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		var warnings uint32 = 0
		if val := tx.Bucket([]byte(maliciousMac)).Get(mac); val != nil {
			warnings = binary.LittleEndian.Uint32(val)
		}
		danger = int64(warnings) >= int64(threshold)
		return nil
	})
	return
}

func RemoveMAC(mac []byte) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(maliciousMac)).Delete(mac)
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

func MarkMAC(mac []byte, threshold int) (danger bool, err error) {
	var tx *bolt.Tx
	var warnings uint32 = 0
	if tx, err = main.Begin(true); err != nil {
		return
	}
	defer tx.Rollback()

	if val := tx.Bucket([]byte(maliciousMac)).Get(mac); val != nil {
		warnings = binary.LittleEndian.Uint32(val)
	}
	warnings += 1
	danger = int64(warnings) >= int64(threshold)
	val := make([]byte, 4)
	binary.LittleEndian.PutUint32(val, warnings)
	if err = tx.Bucket([]byte(maliciousMac)).Put(mac, val); err != nil {
		danger = false
	}
	// Commit the transaction and check for error.
	err = tx.Commit()
	return
}
