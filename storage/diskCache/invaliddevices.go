package diskCache

import (
	"encoding/binary"
	"fmt"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
	"time"
)

func MarkInvalidDevice(mac []byte, maxInterval int) (reject, newDevice bool, err error) {
	var tx *bolt.Tx
	var originalTS int64 = 0
	nowTS := time.Now().Unix()
	if tx, err = main.Begin(true); err != nil {
		return
	}
	defer tx.Rollback()

	if val := tx.Bucket([]byte(invalidDevices)).Get(mac); val != nil {
		// device was already flagged
		originalTS = int64(binary.LittleEndian.Uint64(val))
		reject = (nowTS - originalTS) > int64(maxInterval)
		newDevice = false
	} else {
		// device was not flagged before
		ts := make([]byte, 8)
		binary.LittleEndian.PutUint64(ts, uint64(nowTS))
		err = tx.Bucket([]byte(invalidDevices)).Put([]byte(mac), []byte(ts))
		reject = false
		newDevice = true
	}

	// Commit the transaction and check for error.
	err = tx.Commit()
	return
}

//func AddInvalidDevice(mac []byte) (err error) {
//	err = main.Update(func(tx *bolt.Tx) error {
//		ts := make([]byte, 8)
//		binary.LittleEndian.PutUint64(ts, uint64(time.Now().Unix()))
//		err = tx.Bucket([]byte(invalidDevices)).Put([]byte(mac), []byte(ts))
//		if err != nil {
//			if globals.DebugActive {
//				fmt.Println(err.Error())
//			}
//			return globals.SensorDBError
//		}
//		return nil
//	})
//	return err
//}

func RemoveInvalidDevice(mac []byte) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(invalidDevices)).Delete([]byte(mac))
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
