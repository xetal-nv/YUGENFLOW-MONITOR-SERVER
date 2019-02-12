package servers

import (
	"countingserver/support"
	"fmt"
	"log"
	"net"
	"strings"
)

// TODO main handler
func handlerTCPRequest(conn net.Conn) {

	buf := make([]byte, 3)
	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]

	if _, e := conn.Read(buf); e != nil {
		log.Printf("handlerTCPRequest: Error reading from %v : %v\n", ipc, e)
	} else {
		gnum := int(buf[1]) | int(buf[0])<<8
		fmt.Println(support.Timestamp(), ",", gnum, int(buf[2]))
	}
	//noinspection GoUnhandledErrorResult
	//conn.Close()
}
