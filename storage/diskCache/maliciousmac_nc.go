// +build newcache

package diskCache

import (
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
	"strconv"
)

func CheckMAC(mac []byte, threshold int) (danger bool, err error) {
	w, found := maliciousMac.Get(string(mac))
	if !found {
		return false, nil
	}
	if warnings, e := strconv.ParseInt(w, 16, 64); e == nil {
		danger = warnings >= int64(threshold)
		return
	} else {
		err = e
		return
	}
}

func RemoveSuspectedMAC(mac []byte) (err error) {
	maliciousMac.Delete(string(mac))
	return
}

func MarkMAC(mac []byte, threshold int) (danger bool, err error) {

	r := func(mac, warn string) (string, string) {
		if warn == "" {
			return mac, "1"
		}
		if warnings, e := strconv.ParseInt(warn, 16, 64); e == nil {
			warnings += 1
			return mac, strconv.FormatInt(warnings, 16)
		} else {
			return mac, "1"
		}
	}

	_, warn, _ := maliciousMac.FunctionUpdate(string(mac), r, jac.DefaultExpiration, true)

	if warnings, e := strconv.ParseInt(warn, 16, 64); e == nil {
		danger = warnings >= int64(threshold)
	} else {
		err = globals.SensorDBError
	}
	return
}
