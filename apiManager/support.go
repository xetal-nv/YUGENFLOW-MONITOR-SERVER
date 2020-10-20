package apiManager

import (
	"encoding/binary"
	"fmt"
	"gateserver/sensorManager"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"strconv"
	"strings"
	"time"
)

// execute a command towards a sensor as specified by the params map
// see commandNames definition for what parameters are allowed
func executeCommand(params map[string]string) (rv JsonCmdRt) {
	if params["cmd"] == "" {
		rv.Error = "syntax error"
		return
	}
	if params["cmd"] == "list" {
		keys := ""
		for k := range sensorManager.CmdAPI {
			keys += k + ", "
		}
		rv.Answer = keys + "list,"
		rv.Error = ""
	} else {
		// id and mac are prepared and connection checked
		var id int
		var mac string
		var eid error
		var ok bool
		id, eid = strconv.Atoi(params["id"])
		if mac, ok = params["mac"]; ok {
			mac = strings.Trim(strings.Replace(mac, ":", "", -1), ":")
		}
		if !ok && eid != nil {
			rv.Error = "syntax error"
			return
		} else if !ok {
			if cMac, err := diskCache.LookUpMac([]byte{byte(id)}); err != nil {
				rv.Error = "error: sensor with id " + strconv.Itoa(id) + " not connected"
				return
			} else {
				ok = true
				mac = cMac
			}
		}

		if active, err := diskCache.ReadDeviceStatus([]byte(mac)); err == nil && active {
			sensorManager.ActiveSensors.RLock()
			channels, ok := sensorManager.ActiveSensors.Mac[string(mac)]
			if !ok {
				rv.Error = "sensor in guru meditation"
				sensorManager.ActiveSensors.RUnlock()
				return
			}
			if params["cmd"] == "setid" {
				//  setid needs to check if all data is given and if the id is available
				if ok && eid == nil {
					if cMac, err := diskCache.LookUpMac([]byte{byte(id)}); err == nil {
						rv.Error = "error: sensor id " + strconv.Itoa(id) + " already assigned to " + cMac
						sensorManager.ActiveSensors.RUnlock()
						return
					}
				} else {
					rv.Error = "syntax error"
					sensorManager.ActiveSensors.RUnlock()
					return
				}
			}
			//fmt.Println(params["cmd"], id, mac, params["val"])

			v, valid := sensorManager.CmdAPI[params["cmd"]]
			if !valid {
				rv.Error = "syntax error"
				sensorManager.ActiveSensors.RUnlock()
				return
			}
			cmd := []byte{v.Cmd}
			bs := make([]byte, 2)
			binary.BigEndian.PutUint16(bs, uint16(id))
			cmd = append(cmd, bs...)
			select {
			case channels.Commands <- cmd:
				select {
				case ans := <-channels.Commands:
					rv.Answer = fmt.Sprintf("% x", ans)
					if ans[0] == sensorManager.CmdAPI[params["cmd"]].Cmd {
						rv.Error = ""
					} else {
						rv.Error = "failed"
					}
				case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
					rv.Error = "failed to receive command answer"
				}
			case <-time.After(time.Duration(globals.SettleTime) * time.Second):
				rv.Error = "failed to send command"
			}
			sensorManager.ActiveSensors.RUnlock()
		} else {
			rv.Error = "error: sensor " + mac + " not active"
			return
		}
	}
	return
}
