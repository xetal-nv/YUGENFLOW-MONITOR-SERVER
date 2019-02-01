package servers

import (
	"context"
	"countingserver/spaces"
	"log"
	"net"
	"os"
)

func StartTCP(sd chan context.Context) {

	spaces.SetUp()
	//spaces.CountersSetpUp()

	// Listen for incoming connections.
	port := os.Getenv("TCPPORT")
	l, e := net.Listen(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	if e != nil {
		log.Fatal("StartTCP: fatal error: ", e)
	}

	go func() {
		<-sd
		//noinspection GoUnhandledErrorResult
		l.Close()
	}()

	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				log.Println("StartTCP: recovering server ", port, " from:\n ", e)
				sd <- context.Background() // close running shutdown goroutine
				//noinspection GoUnhandledErrorResult
				l.Close()
				StartTCP(sd)
			}
		}
	}()

	log.Printf("StartTCP: listening on 0.0.0.0:%v\n", port)
	for {
		// Listen for an incoming connection.
		conn, e := l.Accept()
		if e != nil {
			log.Printf("startHTTP: Error accepting: %v\n", e)
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go testHandlerTCPRequest(conn)
	}
}
