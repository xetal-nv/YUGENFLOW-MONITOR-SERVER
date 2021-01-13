// +build newcache

package diskCache

import (
	"encoding/json"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
)

func ReadDefinition(mac []byte) (dr dataformats.SensorDefinition, err error) {
	if msg, found := definitions.Get(string(mac)); found {
		err = json.Unmarshal([]byte(msg), &dr)
	} else {
		err = globals.SensorDBError
	}
	return
}

func WriteDefinition(mac []byte, dr dataformats.SensorDefinition) error {
	bdr, err := json.Marshal(dr)
	if err != nil {
		return err
	}
	definitions.Set(string(mac), string(bdr), jac.DefaultExpiration, true)
	return err
}

func DeleteDefinition(mac []byte) (err error) {
	definitions.Delete(string(mac))
	return
}

func ReadAllDefinitions() (dr []dataformats.SensorDefinition, err error) {
	for key, def := range definitions.Items() {
		var definition dataformats.SensorDefinition
		if e := json.Unmarshal([]byte(def), &definition); e != nil && err == nil {
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
	return
}
