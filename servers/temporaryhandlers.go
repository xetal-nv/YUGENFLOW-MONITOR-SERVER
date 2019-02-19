package servers

import (
	"countingserver/spaces"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

//var counter int
//var once sync.Once

// test Handler
func tempHTTPfuncHandler(message string) http.Handler {
	m := message
	log.Println("Test Handler: started")
	if rand.Intn(5) == 2 {
		panic("setup error")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("Test tempHTTPfuncHandler: recovering from: ", e)
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
func tempHandlerTCPRequest(conn net.Conn) {

	//once.Do(func() { counter = 0 })
	// Buffer is fixed in size as from specs
	buf := make([]byte, 3)
	ipc := strings.Split(conn.RemoteAddr().String(), ":")[0]

	if _, e := conn.Read(buf); e != nil {
		log.Printf("tempHandlerTCPRequest: Error reading from %v : %v\n", ipc, e)
	} else {
		gnum := int(buf[1]) | int(buf[0])<<8

		// DEBUG
		//fmt.Println(support.Timestamp(), ",", int(buf[2]))

		if e := spaces.SendData(gnum, int(buf[2])); e != nil {
			log.Println(e)
		}

	}
	//noinspection GoUnhandledErrorResult
	conn.Close()
}

// Test handler2
func tempHandlerTCPRequest2(conn net.Conn, f bool) {

	if f {
		fmt.Println("New connection arrived")
		go func() {
			time.Sleep(3 * time.Second)
			_, _ = conn.Write([]byte("\x06"))
		}()
	}

	buf := make([]byte, 1)
	if _, e := conn.Read(buf); e == nil {
		fmt.Printf("%v ", buf)
	} else {
		fmt.Printf("error: %v\n", e)
	}
	tempHandlerTCPRequest2(conn, false)
}
