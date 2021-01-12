// +build newcache

package diskCache

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
	"strconv"
	"time"
)

func MarkInvalidDevice(mac []byte, maxInterval int) (reject, newDevice bool, err error) {
	nowTS := time.Now().Unix()
	recordedTS, found := invalidDevices.Add(string(mac), strconv.FormatInt(nowTS, 16),
		jac.DefaultExpiration, true)
	if found {
		// device was already flagged
		if originalTS, e := strconv.ParseInt(recordedTS, 16, 64); e == nil {
			reject = (nowTS - originalTS) > int64(maxInterval)
			newDevice = false
		} else {
			fmt.Println(e)
			err = globals.SensorDBError
		}
	} else {
		// device was not flagged before
		reject = false
		newDevice = true
	}
	return
}

func RemoveInvalidDevice(mac []byte) (err error) {
	invalidDevices.Delete(string(mac))
	return
}

func ListInvalidDevices() (macs []string, tss []int64, err error) {
	for mac, recordedTS := range invalidDevices.Items() {
		if ts, e := strconv.ParseInt(recordedTS, 16, 64); e == nil {
			tss = append(tss, ts)
			macs = append(macs, mac)
		}
	}
	return
}
