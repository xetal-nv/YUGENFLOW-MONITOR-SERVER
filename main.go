package main

import (
	"flag"
	"fmt"
	"gateserver/gateManager"
	"gateserver/sensorManager"
	"gateserver/sensormodels"
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
	var eeprom = flag.Bool("eeprom", false, "enable sensor eeprom refresh at every connection")
	var tcpdeadline = flag.Int("tdl", 24, "TCP read deadline in hours (default 24)")

	flag.Parse()
	globals.DebugActive = *debug
	globals.TCPdeadline = *tcpdeadline
	globals.SensorEEPROMResetEnabled = *eeprom

	fmt.Printf("\nStarting server YugenFlow Server %s \n\n", globals.VERSION)
	if *debug {
		fmt.Println("*** WARNING: Debug mode enabled ***")
	}
	if *tcpdeadline != 24 {
		fmt.Printf("*** WARNING: TCP deadline set to non standard value %v ***\n", globals.TCPdeadline)
	}
	if *eeprom {
		fmt.Printf("*** WARNING: sensor EEPROM refresh enabled ***\n")
	}

	globals.Start()

	// setup shutdown procedure
	c := make(chan os.Signal, 0)
	var sd []chan bool
	for i := 0; i < 2; i++ {
		sd = append(sd, make(chan bool))
	}

	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func(c chan os.Signal, sd []chan bool) {
		<-c
		fmt.Println("\nClosing YugenFlow Server")
		var wg sync.WaitGroup
		for _, ch := range sd {
			wg.Add(1)
			go func(ch chan bool) {
				ch <- true
				select {
				case <-ch:
				case <-time.After(time.Duration(globals.ShutdownTime) * time.Second):
				}
				wg.Done()
			}(ch)
		}
		wg.Wait()
		time.Sleep(2 * time.Second)
		fmt.Println("Closing YugenFlow Server completed")
		os.Exit(0)
	}(c, sd)

	if globals.DebugActive {
		go sensormodels.SensorModel(1, 7000, 10, []int{-1, 1}, []byte{'a', 'b', 'c', '1', '2', '1'})
	}

	go sensorManager.Start(sd[0])
	gateManager.Start(sd[1])
}
