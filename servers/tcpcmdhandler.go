package servers

import (
	"encoding/binary"
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
	//mutexSensorMacs.RLock()
	if params["cmd"] != "" || params["id"] != "" {
		if params["cmd"] == "list" {
			keys := ""
			for k := range cmdAPI {
				keys += k + ", "
			}
			rv.Rt = keys + "list, macid"
			rv.State = true
		} else {
			id, eid := strconv.Atoi(params["id"])
			mace := params["mac"]
			if eid == nil || mace != "" {
				if params["cmd"] == "macid" {
					var mac []byte
					if c, e := net.ParseMAC(params["val"]); e == nil {
						for _, v := range c {
							mac = append(mac, v)
						}
						mutexUnknownMac.RLock()
						if ch, ok := unknownMacChan[string(mac)]; ok {
							mutexUnknownMac.RUnlock()
							mutexSensorMacs.RLock()
							if oldMac, ok := sensorMacID[id]; ok {
								mutexSensorMacs.RUnlock()
								rv.Rt = "error: id assigned to " + string(oldMac)
							} else {
								mutexSensorMacs.RUnlock()
								select {
								case ch <- nil:
									select {
									case conn := <-ch:
										v, _ := cmdAPI["setid"]
										cmd := []byte{v.cmd}
										bs := make([]byte, 2)
										binary.BigEndian.PutUint16(bs, uint16(id))
										cmd = append(cmd, bs...)
										cmd = append(cmd, codings.Crc8(cmd))
										if _, err := conn.Write(cmd); err != nil {
											rv.Rt = "error: command failed"
										} else {
											rv.Rt = "Device restarting"
											rv.State = true
											mutexUnknownMac.Lock()
											unkownDevice[string(mac)] = true
											mutexUnknownMac.Unlock()
											// read and discard answer
											c := make(chan bool)
											go func(c chan bool) {
												_, _ = conn.Read(make([]byte, 256))
												c <- true
											}(c)
											select {
											case <-c:
											case <-time.After(time.Duration(timeout) * time.Second):
											}
										}
										select {
										case ch <- nil:
										case <-time.After(time.Duration(timeout) * time.Second):
											rv.Rt = "warning: command probably failed"
										}
									case <-time.After(time.Duration(timeout) * time.Second):
										rv.Rt = "error: command failed"
									}
								case <-time.After(time.Duration(timeout) * time.Second):
									rv.Rt = "error: command failed"
								}
							}
						} else {
							mutexUnknownMac.RUnlock()
							mutexSensorMacs.RLock()
							if v, ok := sensorIdMAC[string(mac)]; ok {
								mutexSensorMacs.RUnlock()
								rv.Rt = "error: mac assigned to " + strconv.Itoa(v)
							} else {
								mutexSensorMacs.RUnlock()
								rv.Rt = "error: absent"
							}

						}
					} else {
						rv.Rt = "error: invalic mac address given"
					}
				} else {
					var ch chan []byte
					var ok bool
					if mace == "" {
						mutexSensorMacs.RLock()
						ch, ok = SensorCmdID[id]
						mutexSensorMacs.RUnlock()
					} else {
						mutexSensorMacs.RLock()
						ok = true
						ch = SensorCmdMac[mace][1]
						mutexSensorMacs.RUnlock()
					}
					if ok {
						if support.Debug != 0 {
							fmt.Println("CMD: found CMD channel")
						}
						if v, ok := cmdAPI[params["cmd"]]; ok {
							if support.Debug != 0 {
								fmt.Println("CMD: accepted CMD", cmdAPI[params["cmd"]])
							}
							var to int
							var e error
							if to, e = strconv.Atoi(params["timeout"]); e != nil || params["timeout"] == "" {
								to = timeout
							}
							cmd := []byte{v.cmd}
							// need to execute the command on sensor with the given ID
							if v.lgt != 0 && params["val"] != "" {
								if support.Debug != 0 {
									fmt.Println("CMD: found PARAMS", params["val"])
								}
								if data, err := strconv.Atoi(strings.Trim(params["val"], " ")); err != nil {
									cmd = nil
									rv.Rt = "wrong parameters"
								} else {
									// check if the command is a setid and if the id is valid
									if params["cmd"] == "setid" {
										mutexSensorMacs.RLock()
										if sensorChanUsedID[data] {
											rv.Rt = "ID already in use"
											cmd = nil
										}
										mutexSensorMacs.RUnlock()
									}
									if cmd != nil {
										bs := make([]byte, 4)
										binary.BigEndian.PutUint32(bs, uint32(data))
										cmd = append(cmd, bs[4-v.lgt:4]...)
									}
								}
							} else if (v.lgt != 0 && params["val"] == "") || (v.lgt == 0 && params["val"] != "") {
								cmd = nil
								rv.Rt = "insufficient parameters"
							}
							if cmd != nil {
								if support.Debug != 0 {
									fmt.Println("CMD: Executing command")
								}
								select {
								case ch <- cmd:
									if support.Debug != 0 {
										fmt.Println("CMD: sent command", cmd)
									}
									rv.State = true
									select {
									case rt := <-ch:
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
		}
	}
	//mutexSensorMacs.RUnlock()
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
		log.Printf("servers.exeBinaryCommand received and executing %v\n", params)
	}
	return exeParamCommand(params)
}

// handles all command received internal from channel CE and interacts with the associated device and
// handlerTCPRequest (via ci channel) for proper execution.
func handlerCommandAnswer(conn net.Conn, ci, ce chan []byte, stop chan bool, devid chan int, id ...int) {
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
				handlerCommandAnswer(conn, ci, ce, stop, devid, id[0])
			} else {
				handlerCommandAnswer(conn, ci, ce, stop, devid)
			}
		}
	}()
	for {
		select {
		case newid := <-devid:
			id = []int{newid}
		case <-ci:
			// unexpected command answer
			go func() {
				support.DLog <- support.DevData{"servers.handlerCommandAnswer device " + strconv.Itoa(id[0]),
					support.Timestamp(), "unsolicited command answer", []int{1}, true}
			}()
			select {
			case ci <- []byte("error"):
			case <-time.After(time.Duration(timeout) * time.Second):
			case <-stop:
			}
		case cmd := <-ce:
			var rt []byte
			// we return nil in case of error
			// verify if the command exists and send it to the device
			if _, ok := cmdAnswerLen[cmd[0]]; ok {
				if support.Debug > 0 {
					fmt.Printf("Received %v by user for device %v\n", cmd, strconv.Itoa(id[0]))
				}
				if cmd[0] == cmdAPI["setid"].cmd {
					if support.Debug > 0 {
						fmt.Printf("Changed id to %v from %v by user\n", int(cmd[2]), strconv.Itoa(id[0]))
					}
					id = []int{int(cmd[2])}
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
