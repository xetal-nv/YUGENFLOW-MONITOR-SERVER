package diskCache

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

func ReadAllDefinitions() (dr []dataformats.SensorDefinition, err error) {
	_ = main.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(definitions)).Cursor()
		for key, def := c.First(); key != nil; key, def = c.Next() {
			var definition dataformats.SensorDefinition
			if e := json.Unmarshal(def, &definition); e != nil && err == nil {
				err = globals.PartialError
			} else {
				mac := string(key)
				for i := 2; i < len(mac); i += 3 {
					mac = mac[:i] + ":" + mac[i:]
				}
				definition.Mac = mac
				dr = append(dr, definition)
			}
		}
		return nil
	})
	return
}
