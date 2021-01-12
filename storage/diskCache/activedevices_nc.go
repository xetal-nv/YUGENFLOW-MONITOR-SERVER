// +build newcache

package diskCache

import (
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
	"strconv"
)

func AddDevice(mac []byte, status bool) error {
	activeDevices.Set(string(mac), strconv.FormatBool(status), jac.DefaultExpiration, true)
	return nil
}

func MarkDeviceNotActive(mac []byte) error {
	return AddDevice(mac, false)
}

func MarkDeviceActive(mac []byte) error {
	return AddDevice(mac, true)
}

func DeleteDevice(mac []byte) error {
	activeDevices.Delete(string(mac))
	return nil
}

func ReadDeviceStatus(mac []byte) (active bool, err error) {
	if data, found := activeDevices.Get(string(mac)); !found {
		err = globals.SensorDBError
	} else {
		if active, err = strconv.ParseBool(data); err != nil {
			err = globals.SensorDBError
		}
	}
	return
}

func ListActiveDevices() (macs []string, status []bool, err error) {
	for mac, active := range activeDevices.Items() {
		var activeFlag bool
		if activeFlag, err = strconv.ParseBool(active); err == nil {
			macs = append(macs, mac)
			status = append(status, activeFlag)
		}
	}
	return
}
