package sensorDB

import (
	"encoding/binary"
	"fmt"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
)

func CheckIP(ip []byte, threshold int) (danger bool, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		var warnings uint32 = 0
		if val := tx.Bucket([]byte(maliciousIp)).Get(ip); val != nil {
			warnings = binary.LittleEndian.Uint32(val)
		}
		danger = int64(warnings) >= int64(threshold)
		return nil
	})
	return
}

func RemoveSuspecteIP(ip []byte) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(maliciousIp)).Delete(ip)
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

func MarkIP(ip []byte, threshold int) (danger bool, err error) {
	var tx *bolt.Tx
	var warnings uint32 = 0
	if tx, err = main.Begin(true); err != nil {
		return
	}
	defer tx.Rollback()

	if val := tx.Bucket([]byte(maliciousIp)).Get(ip); val != nil {
		warnings = binary.LittleEndian.Uint32(val)
	}
	warnings += 1
	danger = int64(warnings) >= int64(threshold)
	val := make([]byte, 4)
	binary.LittleEndian.PutUint32(val, warnings)
	if err = tx.Bucket([]byte(maliciousIp)).Put(ip, val); err != nil {
		danger = false
	}
	// Commit the transaction and check for error.
	err = tx.Commit()
	return
}
