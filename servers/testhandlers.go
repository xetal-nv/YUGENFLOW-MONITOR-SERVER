package servers

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"playground/spaces"
	"strings"
)

//var counter int
//var once sync.Once

// test Handler
func testHTTPfuncHandler(message string) http.Handler {
	m := message
	log.Println("Test Handler: started")
	if rand.Intn(5) == 2 {
		panic("setup error")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("Test testHTTPfuncHandler: recovering from: ", e)
					//noinspection GoUnhandledErrorResult
					fmt.Fprintf(w, "Good try :-)")
				}
			}
		}()
		//noinspection GoUnhandledErrorResult
		fmt.Fprintf(w, m)
		if m == "" {
			panic("panic address")
		}
	})
}

// Test handler
func testHandlerTCPRequest(conn net.Conn) {

	//once.Do(func() { counter = 0 })
	// Buffer is fixed in size as from specs
	buf := make([]byte, 3)
	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]

	if _, e := conn.Read(buf); e != nil {
		log.Printf("testHandlerTCPRequest: Error reading from %v : %v\n", ipc, e)
	} else {
		gnum := int(buf[1]) | int(buf[0])<<8
		//if buf[2] != 0 && buf[2] != 255 && buf[2] != 1 {
		//	log.Printf("testHandlerTCPRequest: illegal value from %v\n", ipc) // do we add a forbidden IP list?
		//} else {
		//	if buf[2] == 255 {
		//		counter -= 1
		//	}
		//	if buf[2] == 1 {
		//		counter += 1
		//	}
		//	fmt.Printf("counter from %v@%v = %v\n", gnum, ipc, counter)
		//}
		//fmt.Println("ECHO from ", conn.RemoteAddr(), " : ", gnum, buf[2])
		if e := spaces.SendData(gnum, int(buf[2])); e != nil {
			log.Println(e)
		}

	}
	//noinspection GoUnhandledErrorResult
	conn.Close()
}
