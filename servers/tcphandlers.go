package servers

import (
	"countingserver/spaces"
	"io"
	"log"
	"net"
	"strings"
)

// TODO main handler
// Will need to use locks for reading/writing or something else (better)
func handlerTCPRequest(conn net.Conn) {

	defer func() {
		//noinspection GoUnhandledErrorResult
		conn.Close()
	}()

	mac := make([]byte, 6)
	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]

	// Initially receive the MAC address
	if _, e := conn.Read(mac); e != nil {
		log.Printf("servers.handlerTCPRequest: Error reading from %v::%v : %v\n", ipc, mac, e)
	} else {
		log.Printf("servers.handlerTCPRequest: New connected device %v::%v\n", ipc, mac)

		// Start reading data
		// TODO make it to handle EOF !!!
		loop := true
		for loop {
			cmd := make([]byte, 1)
			if _, e := conn.Read(cmd); e != nil {

				if e == io.EOF {
					loop = false
					log.Printf("servers.handlerTCPRequest: connection lost with device %v::%v\n", ipc, mac)
				} else {
					log.Printf("servers.handlerTCPRequest: Error reading from %v::%v : %v\n", ipc, mac, e)
				}
			} else {
				// Take action depending on the received data
				switch cmd[0] {
				case 1:
					// Gate new counting data
					data := make([]byte, 3)
					if _, e := conn.Read(data); e != nil {
						log.Printf("servers.handlerTCPRequest: Error reading from %v::%v : %v\n", ipc, mac, e)
					} else {
						// Valid data
						gid := int(data[1]) | int(data[0])<<8
						if e := spaces.SendData(gid, int(data[2])); e != nil {
							log.Println(e)
						}
						//fmt.Printf("%v :: ID %v :: VALUE %v\n", support.Timestamp(), gid, int8(data[2]))
					}
				default:
					log.Printf("servers.handlerTCPRequest: received illegal command %v from %v::%v",
						int(cmd[0]), ipc, mac)
				}
			}
		}
	}
}
