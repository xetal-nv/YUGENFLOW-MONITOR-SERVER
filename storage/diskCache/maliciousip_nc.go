// +build newcache

package diskCache

import (
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
	"strconv"
)

func CheckIP(ip []byte, threshold int) (danger bool, err error) {
	w, found := maliciousIp.Get(string(ip))
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

func RemoveSuspectedIP(ip []byte) (err error) {
	maliciousIp.Delete(string(ip))
	return
}

func MarkIP(ip []byte, threshold int) (danger bool, err error) {

	r := func(ip, warn string) (string, string) {
		if warn == "" {
			return ip, "1"
		}
		if warnings, e := strconv.ParseInt(warn, 16, 64); e == nil {
			warnings += 1
			return ip, strconv.FormatInt(warnings, 16)
		} else {
			return ip, "1"
		}
	}

	_, warn, _ := maliciousIp.FunctionUpdate(string(ip), r, jac.DefaultExpiration, true)

	if warnings, e := strconv.ParseInt(warn, 16, 64); e == nil {
		danger = warnings >= int64(threshold)
	} else {
		err = globals.SensorDBError
	}
	return
}
