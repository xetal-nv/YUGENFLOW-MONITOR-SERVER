package apiManager

import (
	"encoding/binary"
	"fmt"
	"gateserver/sensorManager"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"strconv"
	"strings"
	"time"
)

// execute a command towards a sensor as specified by the params map
// see commandNames definition for what parameters are allowed
func executeCommand(params map[string]string) (rv JsonCmdRt) {
	var locked bool = false // used toi make sure lock is removed in all possible paths
	defer func() {
		if locked {
			sensorManager.ActiveSensors.RUnlock()
		}
		if r := recover(); r != nil {
			rv.Error = globals.SyntaxError.Error()
			mlogger.Recovered(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.executeCommand",
					"service terminated unexpectedly",
					[]int{1}, true})
		}
	}()
	if params["cmd"] == "" {
		rv.Error = globals.SyntaxError.Error()
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
		if params["cmd"] == "setid" && (!ok || eid != nil) {
			rv.Error = globals.SyntaxError.Error()
			return
		}
		if params["cmd"] != "setid" && ok && eid == nil {
			rv.Error = globals.SyntaxError.Error()
			return
		}

		if !ok {
			if cMac, err := diskCache.LookUpMac([]byte{byte(id)}); err != nil {
				rv.Error = "error: sensor with id " + strconv.Itoa(id) + " not connected"
				return
			} else {
				ok = true
				mac = cMac
			}
		}

		cmdSpecs, valid := sensorManager.CmdAPI[params["cmd"]]
		if !valid {
			rv.Error = globals.SyntaxError.Error()
			return
		}
		param, paramFound := strconv.Atoi(params["val"])
		if cmdSpecs.Lgt != 0 && paramFound != nil {
			rv.Error = globals.SyntaxError.Error()
			return
		}

		if active, err := diskCache.ReadDeviceStatus([]byte(mac)); err == nil && active {
			sensorManager.ActiveSensors.RLock()
			locked = true
			channels, ok := sensorManager.ActiveSensors.Mac[string(mac)]
			if !ok {
				rv.Error = "sensor in guru meditation"
				return
			}
			if params["cmd"] == "setid" {
				//  setid needs to check if all data is given and if the id is available
				if ok && eid == nil {
					if cMac, err := diskCache.LookUpMac([]byte{byte(id)}); err == nil {
						rv.Error = "error: sensor id " + strconv.Itoa(id) + " already assigned to " + cMac
						return
					}
				} else {
					rv.Error = globals.SyntaxError.Error()
					return
				}
			}
			cmd := []byte{cmdSpecs.Cmd}

			switch cmdSpecs.Lgt {
			case 2:
				if paramFound != nil {
					rv.Error = globals.SyntaxError.Error()
					return
				}
				data := make([]byte, 2)
				binary.BigEndian.PutUint16(data, uint16(param))
				cmd = append(cmd, data...)
			case 1:
				par, err := strconv.Atoi(params["val"])
				if err != nil {
					rv.Error = globals.SyntaxError.Error()
					return
				}
				cmd = append(cmd, byte(par))
			default:
			}

			select {
			case channels.Commands <- cmd:
				select {
				case ans := <-channels.Commands:
					if ans != nil {
						rv.Answer = fmt.Sprintf("% x", ans)
						if ans[0] == sensorManager.CmdAPI[params["cmd"]].Cmd {
							rv.Error = ""
						} else {
							rv.Error = globals.Error.Error()
						}
					} else {
						rv.Error = globals.Error.Error()
					}
				case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
					rv.Error = globals.Error.Error()
				}
			case <-time.After(time.Duration(globals.SettleTime) * time.Second):
				rv.Error = globals.Error.Error()
			}
		} else {
			rv.Error = "error: sensor with mac " + mac + " not active"
			return
		}
	}
	return
}
