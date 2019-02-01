package servers

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"
)

// StartServers starts all required HTTP/TCP servers
func StartServers() {

	// TODO add TCP server

	c1 := make(chan bool)      // error quit signal
	c2 := make(chan os.Signal) // quit signal
	ready := false             // it is needed to avoid hanging on c1 before reaching the termination fork

	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				log.Println("StartServers: recovering from ", e)
				// terminating all running servers
				for _, v := range sdServer {
					if v != nil {
						v <- context.Background()
					}
				}
				// terminating the current StartServers
				if ready {
					c1 <- true
				}
				StartServers()
			}
		}
	}()

	if e := setup(); e != nil {
		log.Println("StartServers: server set-up error: ", e)
	} else {

		for i := range addServer {
			// Start HTTP servers
			sdServer[i] = make(chan context.Context)
			startHTTP(addServer[i], sdServer[i], hMap[i])
		}

		// Two way termination to handle:
		// -  Graceful shutdown when quit via SIGINT (Ctrl+C)
		//    SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
		// - error termination and restart

		signal.Notify(c2, os.Interrupt)
		ready = true
		select {
		case <-c1: // error reported elsewhere, need terminating
		case <-c2: // user termination
			<-c2
			log.Println("StartServers: shutting down")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// Signal shutdown to active servers
			for _, v := range sdServer {
				v <- ctx
			}
			os.Exit(0)
		}
	}
}
