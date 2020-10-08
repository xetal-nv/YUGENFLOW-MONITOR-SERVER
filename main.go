package main

import (
	"flag"
	"fmt"
	"gateserver/entryManager"
	"gateserver/gateManager"
	"gateserver/sensorManager"
	"gateserver/sensormodels"
	"gateserver/spaceManager"
	"gateserver/storage/coredbs"
	"gateserver/storage/sensorDB"
	"gateserver/support/globals"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// to be split back-end, API and web server
func main() {
	var dbpath = flag.String("db", "mongodb://localhost:27017", "database path")
	var dcpath = flag.String("dc", "tables", "2nd level cache disk path")
	//var de = flag.Bool("dumpentry", false, "dump all entry data to log files")
	var debug = flag.Bool("debug", false, "enable debug mode")
	var dev = flag.Bool("dev", false, "to be removed")
	var eeprom = flag.Bool("eeprom", false, "enable sensor eeprom refresh at every connection")
	var tcpdeadline = flag.Int("tdl", 24, "TCP read deadline in hours (default 24)")
	var failTh = flag.Int("fth", 3, "failure threshold in severe mode (default 3)")
	var user = flag.String("user", "", "user name")
	var pwd = flag.String("pwd", "", "user password")

	flag.Parse()
	globals.DebugActive = *debug
	globals.TCPdeadline = *tcpdeadline
	globals.SensorEEPROMResetEnabled = *eeprom
	globals.DiskCachePath = *dcpath
	globals.FailureThreshold = *failTh
	globals.DBpath = *dbpath
	globals.DBUser = *user
	globals.DBUserPassword = *pwd

	//globals.LogToFileAll = *de

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
	//if *de {
	//	log.Printf("*** WARNING: dump all data is enabled ***\n")
	//}
	fmt.Printf("*** INFO: failure threshold set to %v ***\n", *failTh)

	if err := coredbs.Start(); err != nil {
		fmt.Println("Failed top start the data database:", err.Error())
		os.Exit(0)
	}
	globals.Start()
	sensorDB.Start()
	sensorManager.LoadSensorEEPROMSettings()

	// setup shutdown procedure
	c := make(chan os.Signal, 0)
	var sd []chan bool
	for i := 0; i < 4; i++ {
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
				case <-time.After(time.Duration(globals.SettleTime) * time.Second):
				}
				wg.Done()
			}(ch)
		}
		sensorDB.Close()
		wg.Wait()
		if err := coredbs.Disconnect(); err != nil {
			fmt.Println("Error in disconnecting from the YugenFlow Database")
		} else {
			fmt.Println("Disconnected from YugenFlow Database")
		}
		time.Sleep(2 * time.Second)
		fmt.Println("Closing YugenFlow Server completed")
		os.Exit(0)
	}(c, sd)

	//goland:noinspection ALL
	go spaceManager.Start(sd[0])
	time.Sleep(time.Duration(globals.SettleTime) * time.Second)
	//goland:noinspection ALL
	go entryManager.Start(sd[1])
	time.Sleep(time.Duration(globals.SettleTime) * time.Second)
	//goland:noinspection ALL
	go sensorManager.Start(sd[2])

	//if globals.DebugActive {
	if *dev {
		go sensormodels.SensorModel(0, 7000, 3, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x01})
		go sensormodels.SensorModel(1, 7000, 10, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x03})
		go sensormodels.SensorModel(2, 7000, 7, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x07})
		go sensormodels.SensorModel(3, 7000, 5, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x08})
		//time.Sleep(3*time.Second)
		//go sensormodels.SensorModel(4, 7000, 10, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x02})
	}

	//goland:noinspection ALL
	gateManager.Start(sd[3])
}
