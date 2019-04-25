package servers

import (
	"encoding/json"
	"fmt"
	"gateserver/codings"
	"gateserver/support"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

// execute a command towards a sensor as specified by the params map
// see cmds definition for what parameters are allowed
func exeParamCommand(params map[string]string) (rv Jsoncmdrt) {
	rv = Jsoncmdrt{"", false}
	if params["cmd"] != "" || params["id"] != "" {
		if params["cmd"] == "list" {
			if support.Debug != 0 {
				//fmt.Println("CMD: LIST")
			}
			keys := ""
			for k := range cmdAPI {
				keys += k + ", "
			}
			rv.Rt = keys + "list, macid"
			rv.State = true
			//params["async"] = "0"
		} else if id, e := strconv.Atoi(params["id"]); e == nil {
			//if support.Debug != 0 {
			//	fmt.Println("CMD: NOT LIST: ", params)
			//}
			if params["cmd"] == "macid" {
				var mac []byte
				if c, e := net.ParseMAC(params["val"]); e == nil {
					for _, v := range c {
						mac = append(mac, v)
					}
				}
				mutexUnknownMac.RLock()
				//if ch, ok := unknownMacChan[params["val"]]; ok {
				if ch, ok := unknownMacChan[string(mac)]; ok {
					mutexUnknownMac.RUnlock()
					mutexSensorMacs.RLock()
					if oldMac, ok := sensorMacID[id]; ok {
						mutexSensorMacs.RUnlock()
						rv.Rt = "error: id assigned to " + string(oldMac)
					} else {
						mutexSensorMacs.RUnlock()
						ch <- nil
						conn := <-ch
						v, _ := cmdAPI["setid"]
						cmd := []byte{v.cmd}
						cmd = append(cmd, byte(id))
						cmd = append(cmd, codings.Crc8(cmd))
						if _, err := conn.Write(cmd); err != nil {
							rv.Rt = "error: command failed"
						} else {
							rv.State = true
						}
						ch <- nil
					}
				} else {
					mutexUnknownMac.RUnlock()
					mutexSensorMacs.RLock()
					//if v, ok := sensorIdMAC[params["val"]]; ok {
					if v, ok := sensorIdMAC[string(mac)]; ok {
						mutexSensorMacs.RUnlock()
						rv.Rt = "error: mac assigned to " + strconv.Itoa(v)
					} else {
						mutexSensorMacs.RUnlock()
						rv.Rt = "error: absent"
					}

				}
			} else if _, ok := SensorCmdID[id]; ok {
				if support.Debug != 0 {
					fmt.Println("CMD: found CMD channel")
				}
				if v, ok := cmdAPI[params["cmd"]]; ok {
					if support.Debug != 0 {
						fmt.Println("CMD: accepted CMD", cmdAPI[params["cmd"]])
					}
					var to int
					if to, e = strconv.Atoi(params["timeout"]); e != nil || params["timeout"] == "" {
						to = timeout
					}
					cmd := []byte{v.cmd}
					// need to execute the command on sensor with the given ID
					if v.lgt != 0 && params["val"] != "" {
						par := strings.Split(params["val"][1:len(params["val"])-1], ",")
						if support.Debug != 0 {
							fmt.Println("CMD: found PARAMS", par)
						}
						if v.lgt == len(par) {
							for _, val := range par {
								if data, err := strconv.Atoi(strings.Trim(val, " ")); err != nil || data > 255 {
									cmd = nil
									break
								} else {
									cmd = append(cmd, byte(data))
								}
							}
						} else {
							cmd = nil
							rv.Rt = "insufficient parameters"
						}
					}
					if cmd != nil {
						if support.Debug != 0 {
							fmt.Println("CMD: Executing command")
						}
						select {
						case SensorCmdID[id] <- cmd:
							if support.Debug != 0 {
								fmt.Println("CMD: sent command", cmd)
							}
							rv.State = true
							select {
							case rt := <-SensorCmdID[id]:
								if support.Debug != 0 {
									fmt.Println("CMD: received", rt)
								}
								rv.Rt = string(rt)
							case <-time.After(time.Duration(to) * time.Second):
								rv.Rt = "to"
								// timeout to be used on the sending side to remove a possible hanging goroutine
							}
						case <-time.After(time.Duration(to) * time.Second):
							rv.Rt = "to"
						}
					}
				}
			}
		}
	}
	return
}

// execute command CMD with parameter val on sensor ID. all values are strings
func exeBinaryCommand(id, cmd string, val []int) Jsoncmdrt {
	params := make(map[string]string)
	for _, i := range cmds {
		params[i] = ""
	}
	if v, e := json.Marshal(val); e != nil {
		return Jsoncmdrt{"", false}
	} else {
		params["val"] = string(v)
	}
	params["cmd"] = cmd
	params["id"] = id
	if support.Debug != 0 {
		log.Printf("exeBinaryCommand received and executing %v\n", params)
	}
	return exeParamCommand(params)
}

// handles all command received internall from channel CE and interacts with the associated device and
// handlerTCPRequest (via ci channel) for proper execution.
func handlerCommandAnswer(conn net.Conn, ci, ce chan []byte, stop chan bool, id ...int) {
	//loop := true
	if len(id) == 0 {
		id = []int{-1}
	}
	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"servers.handlerCommandAnswer: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			if len(id) == 1 {
				handlerCommandAnswer(conn, ci, ce, stop, id[0])
			} else {
				handlerCommandAnswer(conn, ci, ce, stop)
			}
		}
	}()
	for {
		select {
		case <-ci:
			// unexpected command answer, illegal situation
			go func() {
				support.DLog <- support.DevData{"handlerCommandAnswer device " + strconv.Itoa(id[0]),
					support.Timestamp(), "unsollcited command answer", []int{1}, true}
			}()
			select {
			case ci <- []byte("error"):
			case <-time.After(time.Duration(timeout) * time.Second):
			case <-stop:
			}
		case cmd := <-ce:
			fmt.Println("CMDCH: readying command", cmd)
			var rt []byte
			// we return nil in case of error
			// verify if the command exists and send it to the device
			if _, ok := cmdAnswerLen[cmd[0]]; ok {
				if support.Debug > 0 {
					fmt.Printf("Received %v from user for device %v\n", cmd, strconv.Itoa(id[0]))
				}
				cmd = append(cmd, codings.Crc8(cmd))
				ready := make(chan bool)
				go func(ba []byte) {
					if _, e := conn.Write(ba); e == nil {
						ready <- true
					} else {
						ready <- false
					}
				}(cmd)
				select {
				case valid := <-ready:
					if valid {
						select {
						case rt = <-ci:
						case <-time.After(time.Duration(timeout) * time.Second):
						case <-stop:
						}
					}
				case <-time.After(time.Duration(timeout) * time.Second):
					// avoid hanging goroutines
					go func() { <-ready }()
				case <-stop:
					// avoid hanging goroutines
					go func() { <-ready }()
				}
			}
			go func() { ce <- rt }()
		case <-stop:
		}
	}
}
