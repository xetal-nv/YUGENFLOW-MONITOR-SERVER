// +build newcache

package diskCache

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
	"strconv"
)

func LookUpMac(id []byte) (mac string, err error) {
	var found bool
	if mac, found = lookup.Get(string(id)); !found || mac == "" {
		err = globals.KeyInvalid
	}
	return
}

func AddLookUp(id []byte, mac string) error {
	lookup.Set(string(id), mac, jac.DefaultExpiration, true)
	return nil
}

func DeleteLookUp(id []byte) error {
	lookup.Delete(string(id))
	return nil
}

func GenerateIdLookUp() (table map[string]int, err error) {
	table = make(map[string]int)
	data := lookup.Items()
	for ids, mac := range data {
		if id, e := strconv.Atoi(ids); e == nil {
			table[string(mac)] = id
		} else {
			fmt.Println(e)
			err = globals.PartialError
		}
	}
	return
}
