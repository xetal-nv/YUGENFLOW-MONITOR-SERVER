// +build !newcache

package diskCache

import (
	"fmt"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
)

func AddDevice(mac []byte, status bool) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		if status {
			err = tx.Bucket([]byte(activeDevices)).Put([]byte(mac), []byte("1"))
		} else {
			err = tx.Bucket([]byte(activeDevices)).Put([]byte(mac), []byte("0"))
		}
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

func MarkDeviceNotActive(mac []byte) error {
	return AddDevice(mac, false)
}

func MarkDeviceActive(mac []byte) error {
	return AddDevice(mac, true)
}

func DeleteDevice(mac []byte) (err error) {
	err = main.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(activeDevices)).Delete([]byte(mac))
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

func ReadDeviceStatus(mac []byte) (active bool, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		res := tx.Bucket([]byte(activeDevices)).Get(mac)
		switch string(res) {
		case "0":
			active = false
		case "1":
			active = true
		default:
			err = globals.SensorDBError
		}
		return nil
	})
	return
}

func ListActiveDevices() (macs []string, status []bool, err error) {
	err = main.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(activeDevices))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			macs = append(macs, string(k))
			if string(v) == "0" {
				status = append(status, false)
			} else {
				status = append(status, true)
			}
			//fmt.Printf("key=%s, value=%s\n", k, v)
		}
		return nil
	})
	return
}
