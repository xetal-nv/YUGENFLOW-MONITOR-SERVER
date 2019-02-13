package servers

import (
	"countingserver/support"
	"fmt"
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
		log.Printf("servers.handlerTCPRequest: Error reading from %v : %v\n", ipc, e)
	} else {
		log.Printf("servers.handlerTCPRequest: New connected device from %v with MAC %v\n", ipc, mac)

		// Start reading data
		for {
			cmd := make([]byte, 1)
			if _, e := conn.Read(cmd); e != nil {
				log.Printf("servers.handlerTCPRequest: Error reading from %v : %v\n", ipc, e)
			} else {
				switch cmd[0] {
				case 1:
					// Gate new counting data
					data := make([]byte, 3)
					if _, e := conn.Read(data); e != nil {
						log.Printf("servers.handlerTCPRequest: Error reading from %v : %v\n", ipc, e)
					} else {
						// Valid data
						// TODO send to the proper thread
						gid := int(data[1]) | int(data[0])<<8
						fmt.Printf("%v :: ID %v :: VALUE %v\n", support.Timestamp(), gid, int8(data[2]))
					}
				default:
					log.Printf("servers.handlerTCPRequest: received illegale command %v\n", int(cmd[0]))
				}
			}
			//gnum := int(buf[1]) | int(buf[0])<<8
			//fmt.Println(support.Timestamp(), ",", gnum, int(buf[2]))
		}
	}
}
