// +build !newcache

package diskCache

import (
	"fmt"
	"gateserver/support/globals"
	bolt "go.etcd.io/bbolt"
)

func LookUpMac(id []byte) (mac string, err error) {
	//fmt.Printf("read lookup: %x\n", id)
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
	//fmt.Printf("add lookup: %x\n", id)
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
	//fmt.Printf("delete lookup: %x\n", id)
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

func GenerateIdLookUp() (table map[string]int, err error) {
	table = make(map[string]int)
	err = main.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(lookup))
		c := b.Cursor()
		for id, mac := c.First(); id != nil; id, mac = c.Next() {
			table[string(mac)] = int(id[0])
		}
		return nil
	})
	return
}
