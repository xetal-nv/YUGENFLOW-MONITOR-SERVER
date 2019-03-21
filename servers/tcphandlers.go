package servers

import (
	"countingserver/codings"
	"countingserver/gates"
	"countingserver/support"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func handlerTCPRequest(conn net.Conn) {

	var deviceId int
	loop := true
	idKnown := false
	stop := make(chan bool)

	defer func() {
		if idKnown {
			stop <- true
		}
		go func() {
			support.DLog <- support.DevData{"handlerTCPRequest recover",
				support.Timestamp(), "", []int{1}, true}
		}()
		//noinspection GoUnhandledErrorResult
		conn.Close()
	}()

	mac := make([]byte, 6)
	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]

	// Initially receive the MAC address
	if _, e := conn.Read(mac); e != nil {
		log.Printf("servers.handlerTCPRequest: error reading from %v::%v : %v\n", ipc, mac, e)
	} else {
		log.Printf("servers.handlerTCPRequest: new connected device %v::%v\n", ipc, mac)

		// Start reading data
		for loop {
			cmd := make([]byte, 1)
			if _, e := conn.Read(cmd); e != nil {

				if e == io.EOF {
					// in case of channel closed (EOF) it gets logged and the handler terminated
					log.Printf("servers.handlerTCPRequest: connection lost with device %v::%v\n", ipc, mac)
				} else {
					log.Printf("servers.handlerTCPRequest: error reading from %v::%v : %v\n", ipc, mac, e)
				}
				loop = false
				log.Printf("servers.handlerTCPRequest: closing TCP channel to %v::%v\n", ipc, mac)
			} else {
				switch cmd[0] {
				case 1:
					// Gate new counting data
					var data []byte
					if crcUsed {
						data = make([]byte, 4)
					} else {
						data = make([]byte, 3)
					}
					if _, e := conn.Read(data); e != nil {
						loop = false
						log.Printf("servers.handlerTCPRequest: error reading from %v::%v : %v\n", ipc, mac, e)
						log.Printf("servers.handlerTCPRequest: closing TCP channel to %v::%v\n", ipc, mac)
					} else {
						// Valid data
						valid := true
						if crcUsed {
							msg := append(cmd, data[:3]...)
							crc := codings.Crc8(msg)
							if crc != data[3] {
								if support.Debug > 0 {
									log.Print("servers.handlerTCPRequest: wrong CRC on received message\n")
								}
								valid = false
							}
						}

						if valid {
							// starts handlerCommandAnswer once wkith the proper ID
							if !idKnown {
								deviceId = int(data[1]) | int(data[0])<<8
								sensorMac[deviceId] = mac
								sensorChan[deviceId] = make(chan []byte)
								SensorCmd[deviceId] = make(chan []byte)
								go handlerCommandAnswer(conn, sensorChan[deviceId], SensorCmd[deviceId], stop, deviceId)
								// TODO HERE
								idKnown = true
							}
							// first sample creates the command channels and handler if it does not exists
							if e := gates.SendData(deviceId, int(data[2])); e != nil {
								log.Println(e)
							}
						}
					}
				default:
					if !idKnown {
						loop = false
					} else {
						// verify it is a command answer, if not closes the TCP channel
						if v, ok := cmdAnswerLen[cmd[0]]; ok {
							if !crcUsed {
								v -= 1
							}
							if v == 0 {
								// this will never happen when CRC8 is used
								//fmt.Printf("Received something else %v\n", cmd)
								sensorChan[deviceId] <- cmd
								// if the answer is incorrect the channel will be closed
								if ans := <-sensorChan[deviceId]; ans != nil {
									loop = false
								}
							} else {
								cmdd := make([]byte, v)
								if _, e := conn.Read(cmdd); e != nil {
									loop = false
									log.Printf("servers.handlerTCPRequest: error reading answer from %v::%v "+
										"for command %v\n", ipc, deviceId, cmd)
									log.Printf("servers.handlerTCPRequest: closing TCP channel to %v::%v\n", ipc, mac)
								} else {
									cmd = append(cmd, cmdd...)
									valid := true
									if crcUsed {
										crc := codings.Crc8(cmd[:len(cmd)-1])
										if crc != cmd[len(cmd)-1] {
											if support.Debug > 0 {
												log.Print("servers.handlerTCPRequest: wrong CRC on received message\n")
											}
											valid = false
										}
									}
									if valid {
										select {
										case sensorChan[deviceId] <- cmd[:len(cmd)-1]:
										case <-time.After(time.Duration(timeout) * time.Second):
											// internal issue, all goroutines will close on time out including the channel
											go func() { sensorChan[deviceId] <- cmd }()
											log.Printf("servers.handlerTCPRequest: hanging operation in sending "+
												"command answer %v to user\n", cmd)
										}
									}
								}
							}
						} else {
							loop = false
						}
					}
					if !loop {
						log.Printf("servers.handlerTCPRequest: illegal command/operation %v sent by %v::%v\n", cmd[0], ipc, mac)
						log.Printf("servers.handlerTCPRequest: closing TCP channel to %v::%v\n", ipc, mac)
					}
				}
			}
		}
	}
}

// TODO to be tested with real device
func handlerCommandAnswer(conn net.Conn, ci, ce chan []byte, stop chan bool, id ...int) {
	loop := true
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
	for loop {
		select {
		case <-ci:
			// unexpected command answer, illegal situation
			go func() {
				support.DLog <- support.DevData{"handlerCommandAnswer device " + strconv.Itoa(id[0]),
					support.Timestamp(), "unsollcited command answer", []int{1}, true}
			}()
			ci <- []byte("error")
		case cmd := <-ce:
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
						}
					}
				case <-time.After(time.Duration(timeout) * time.Second):
					// avoid hanging goroutines
					go func() { <-ready }()
				}
			}
			go func() { ce <- rt }()
		case <-stop:
			loop = false
			if support.Debug > 0 {
				fmt.Printf("Received termination signal, device %v\n", strconv.Itoa(id[0]))
			}
		}
	}
}

//func handlerReset(id int) {
//	defer func() {
//		if e := recover(); e != nil {
//			go func() {
//				support.DLog <- support.DevData{"servers.handlerReset: recovering server",
//					support.Timestamp(), "", []int{1}, true}
//			}()
//			handlerReset(id)
//		}
//	}()
//	done := false
//	for {
//		time.Sleep(30 * time.Minute)
//		if skip, e := support.InClosureTime(spaces.SpaceTimes[spacename].Start,spaces.SpaceTimes[spacename].End); e == nil
//	}
//}
