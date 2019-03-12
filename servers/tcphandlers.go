package servers

import (
	"countingserver/codings"
	"countingserver/gates"
	"countingserver/support"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
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
								sensorChan[deviceId] = make(chan []byte, cmdBuffLen)
								SensorCmd[deviceId] = make(chan []byte, cmdBuffLen)
								go handlerCommandAnswer(conn, sensorChan[deviceId], SensorCmd[deviceId], stop, deviceId)
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
										//fmt.Printf("Received something else %v\n", cmd)
										sensorChan[deviceId] <- cmd
										// if the answer is incorrect the channel will be closed
										if ans := <-sensorChan[deviceId]; ans != nil {
											loop = false
										}
									}
								}
							}
						} else {
							loop = false
						}
					}
					if !loop {
						log.Printf("servers.handlerTCPRequest: illegal command %v sent by %v::%v\n", cmd[0], ipc, mac)
						log.Printf("servers.handlerTCPRequest: closing TCP channel to %v::%v\n", ipc, mac)
					}
				}
			}
		}
	}
}

// TODO command handler as well the API channels
func handlerCommandAnswer(conn net.Conn, ci, ce chan []byte, stop chan bool, id ...int) {
	loop := true
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				if len(id) == 1 {
					handlerCommandAnswer(conn, ci, ce, stop, id[0])
				} else {
					handlerCommandAnswer(conn, ci, ce, stop)
				}
			}
		}
	}()
	for loop {
		select {
		case d := <-ci:
			fmt.Printf("Received from device %v\n", d)
			ci <- nil
		case d := <-ce:
			fmt.Printf("Received from user %v\n", d)
		case <-stop:
			loop = false
			fmt.Printf("Received termination signal\n")
		}
	}
}
