package servers

import (
	"context"
	"gateserver/support"
	"log"
	"net"
	"os"
)

// start of the TCP server, including set-up
func StartTCP(sd chan context.Context) {

	setUpTCP()

	// Listen for incoming connections.
	port := os.Getenv("TCPPORT")
	l, e := net.Listen(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	if e != nil {
		log.Fatal("servers.StartTCP: fatal error:", e)
	}

	r := func() {
		<-sd
		//noinspection GoUnhandledErrorResult
		l.Close()
	}

	go support.RunWithRecovery(r, nil)

	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"servers.StartTCP: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			log.Println("servers.StartTCP: recovering server", port, "from:\n", e)
			sd <- context.Background() // close running shutdown goroutine
			//noinspection GoUnhandledErrorResult
			l.Close()
			StartTCP(sd)
		}
	}()

	log.Printf("servers.StartTCP: listening on 0.0.0.0:%v\n", port)
	for {
		//select {
		//case <-tcpTokens:
		//if support.Debug != 0 {
		//	log.Println("Reserved TCP token, remaining:", len(tcpTokens))
		//}
		// Listen for an incoming connection.
		conn, e := l.Accept()
		if e != nil {
			log.Printf("servers.StartTCP: Error accepting: %v\n", e)
			if l != nil {
				_ = l.Close()
			}
		}
		// Handle connections in a new goroutine.
		log.Printf("servers.StartTCP: A device has connected.\n")
		go handlerTCPRequest(conn)
		//default:
		//	support.DLog <- support.DevData{"servers.StartTCP: exceeding number of allowed connections",
		//		support.Timestamp(), "", []int{1}, true}
		//	r := rand.Intn(minDelayRefusedConnection)
		//	time.Sleep(time.Duration(minDelayRefusedConnection+r) * time.Second)
		//}

	}
}
