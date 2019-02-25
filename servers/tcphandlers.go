package servers

import (
	"countingserver/gates"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func handlerTCPRequest(conn net.Conn) {

	//noinspection GoUnhandledErrorResult
	defer conn.Close()

	mac := make([]byte, 6)
	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]

	// Initially receive the MAC address
	if _, e := conn.Read(mac); e != nil {
		log.Printf("servers.handlerTCPRequest: error reading from %v::%v : %v\n", ipc, mac, e)
	} else {
		log.Printf("servers.handlerTCPRequest: new connected device %v::%v\n", ipc, mac)

		// Start reading data
		loop := true
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
						deviceId := int(data[1]) | int(data[0])<<8
						if e := gates.SendData(deviceId, int(data[2])); e != nil {
							log.Println(e)
						}
					}
				default:
					// verify it is a command answer, if not closes the TCP channel
					if v, ok := cmdlen[cmd[0]]; ok {
						if !crcUsed {
							if v -= 1; v == 0 {
								cmdchan <- cmd
							} else {
								cmdd := make([]byte, v)
								if _, e := conn.Read(cmdd); e != nil {
									loop = false
									log.Printf("servers.handlerTCPRequest: error reading answer from %v::%v for command %v\n", ipc, mac, cmd)
									log.Printf("servers.handlerTCPRequest: closing TCP channel to %v::%v\n", ipc, mac)
								} else {
									cmdchan <- append(cmd, cmdd...)
								}
							}
						}
					} else {
						loop = false
						log.Printf("servers.handlerTCPRequest: illegal command %v sent by %v::%v\n", cmd[0], ipc, mac)
						log.Printf("servers.handlerTCPRequest: closing TCP channel to %v::%v\n", ipc, mac)
					}
				}
			}
		}
	}
}

// TODO command handler as well the API channels
func handlerCommandAnswer(c chan []byte) {
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				handlerCommandAnswer(c)
			}
		}
	}()
	for {
		fmt.Printf("Received something else %v\n", <-c)
	}
}
