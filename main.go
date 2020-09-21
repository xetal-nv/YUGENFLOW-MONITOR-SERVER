package main

import (
	"flag"
	"fmt"
	"gateserver/gateManager"
	"gateserver/support/globals"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// in progress
func main() {
	var debug = flag.Bool("debug", false, "enable debug mode")
	flag.Parse()
	globals.DebugActive = *debug

	globals.Start()
	fmt.Println("\nStarting server YugenFlow Server", globals.VERSION)
	if globals.DebugActive {
		fmt.Println("*** WARNING: Debug mode enabled ***")
	}

	// setup shutdown procedure
	c := make(chan os.Signal, 0)
	var sd []chan bool
	for i := 0; i < 1; i++ {
		sd = append(sd, make(chan bool))
	}

	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func(c chan os.Signal, sd []chan bool) {
		<-c
		fmt.Println("Closing YugenFlow Server")
		var wg sync.WaitGroup
		for _, ch := range sd {
			wg.Add(1)
			go func(ch chan bool) {
				ch <- true
				select {
				case <-ch:
				case <-time.After(2 * time.Second):
				}
				wg.Done()
			}(ch)
		}
		wg.Wait()
		time.Sleep(2 * time.Second)
		fmt.Println("Closing YugenFlow Server completed")
		os.Exit(0)
	}(c, sd)

	gateManager.Start(sd[0])
}
