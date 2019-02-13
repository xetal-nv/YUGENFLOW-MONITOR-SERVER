package servers

import (
	"context"
	"log"
	"net"
	"os"
)

//func setUpTCP() {
func SetUpTCP() {
	if os.Getenv("CRC") == "1" {
		crcUsed = true
	} else {
		crcUsed = false
	}

	log.Println("servers.StartTCP: CRC usage is set to", crcUsed)
}
func StartTCP(sd chan context.Context) {

	// Listen for incoming connections.
	port := os.Getenv("TCPPORT")
	l, e := net.Listen(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	if e != nil {
		log.Fatal("servers.StartTCP: fatal error:", e)
	}

	go func() {
		<-sd
		//noinspection GoUnhandledErrorResult
		l.Close()
	}()

	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				log.Println("servers.StartTCP: recovering server", port, "from:\n", e)
				sd <- context.Background() // close running shutdown goroutine
				//noinspection GoUnhandledErrorResult
				l.Close()
				StartTCP(sd)
			}
		}
	}()

	log.Printf("servers.StartTCP: listening on 0.0.0.0:%v\n", port)
	for {
		// Listen for an incoming connection.
		conn, e := l.Accept()
		if e != nil {
			log.Printf("servers.StartTCP: Error accepting: %v\n", e)
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handlerTCPRequest(conn)
		//go tempHandlerTCPRequest(conn)
		//go tempHandlerTCPRequest2(conn, true)
	}
}
