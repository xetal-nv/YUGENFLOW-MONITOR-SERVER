// +build dev,!debug,!script

package main

import (
	"flag"
	"fmt"
	"gateserver/apiManager"
	"gateserver/avgsManager"
	"gateserver/entryManager"
	"gateserver/exportManager"
	"gateserver/gateManager"
	"gateserver/sensorManager"
	"gateserver/sensormodels"
	"gateserver/spaceManager"
	"gateserver/storage/coredbs"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

func main() {
	var dbpath = flag.String("db", "mongodb://localhost:27017", "database path")
	var dcpath = flag.String("dc", "", "2nd level cache disk path")
	var debug = flag.Bool("debug", false, "enable debug mode")
	var delogs = flag.Bool("delogs", false, "delete all logs")
	var dev = flag.Bool("dev", false, "development mode")
	var echo = flag.Int("echo", 0, "enables echo mode (0: off, 1: raw, 2: gates, 3: spaces)")
	var eeprom = flag.Bool("eeprom", false, "enable sensor eeprom refresh at every connection")
	var export = flag.Bool("export", false, "enable export scripting")
	var tcpdeadline = flag.Int("tdl", 24, "TCP read deadline in hours (default 24)")
	var failTh = flag.Int("fth", 3, "failure threshold in severe mode (default 3)")
	var user = flag.String("user", "", "user name")
	var pwd = flag.String("pwd", "", "user password")
	var stress = flag.Int("stress", -1, "stress test cycles (override dev)")
	//var raw = flag.Bool("raw", false, "enable raw mode")
	var st = flag.String("start", "", "set start time expressed as HH:MM")
	var us = flag.Bool("us", false, "enable unsafe shutdown")
	var verbose = flag.Bool("v", false, "enable verbose mode")
	var ll = flag.Bool("ll", false, "legacy unrolled logging")

	flag.Parse()

	if *delogs {
		files, err := filepath.Glob(filepath.Join("log", "*"))
		if err != nil {
			fmt.Printf("*** ERROR: Error while removing logs %v ***\n", err.Error())
		}
		for _, file := range files {
			err = os.RemoveAll(file)
			if err != nil {
				fmt.Printf("*** ERROR: Error while removing logs %v ***\n", err.Error())
			}
		}
	}

	if *st != "" {
		if ns, err := time.Parse(globals.TimeLayout, *st); err != nil {
			fmt.Println("Syntax error in specified start time", *st)
			os.Exit(1)
		} else {
			now := time.Now()
			nows := strconv.Itoa(now.Hour()) + ":"
			mins := "00" + strconv.Itoa(now.Minute())
			nows += mins[len(mins)-2:]
			if ne, err := time.Parse(globals.TimeLayout, nows); err != nil {
				fmt.Println("Syntax error retrieving system current time")
				os.Exit(1)
			} else {
				del := ns.Sub(ne)
				if del > 0 {
					fmt.Println("*** INFO : Waiting", del, "before starting server ***")
					time.Sleep(del)
				} else {
					fmt.Println("*** WARNING: cannot wait in the past ***")
				}
			}
		}
	}

	globals.DebugActive = *debug
	globals.TCPdeadline = *tcpdeadline
	globals.SensorEEPROMResetEnabled = *eeprom
	globals.DiskCachePath = *dcpath
	globals.FailureThreshold = *failTh
	globals.DBpath = *dbpath
	globals.DBUser = *user
	globals.DBUserPassword = *pwd
	globals.EchoMode = *echo == 1
	globals.GateMode = *echo == 2
	globals.SpaceMode = *echo == 3
	globals.ExportEnabled = *export
	globals.LimitedApi = false
	mlogger.Verbose(*verbose)
	mlogger.Unroll(*ll, "log.txt")

	fmt.Printf("\nStarting server YugenFlow Server %s-development \n\n", globals.VERSION)
	if *debug {
		fmt.Println("*** WARNING: Debug mode enabled ***")
	}
	if *tcpdeadline != 24 {
		fmt.Printf("*** WARNING: TCP deadline set to non standard value %v ***\n", globals.TCPdeadline)
	}
	if *eeprom {
		fmt.Printf("*** WARNING: sensor EEPROM refresh enabled ***\n")
	}
	if *export {
		fmt.Println("*** WARNING: Export mode enabled ***")
	}
	fmt.Printf("*** INFO: failure threshold set to %v ***\n", *failTh)
	if *us {
		fmt.Println("*** WARNING: Enabled unsafe shutdown on user signals ***")
	}
	if *echo != 0 {
		fmt.Printf("*** INFO: Echo mode %v enabled ***\n", *echo)
	}
	if *verbose {
		fmt.Println("*** INFO: Running in Verbose mode ***")
	}

	globals.Start()
	diskCache.Start()
	if err := coredbs.Start(); err != nil {
		fmt.Println("Failed top start the data database:", err.Error())
		os.Exit(0)
	}
	sensorManager.LoadSensorEEPROMSettings()

	// setup shutdown procedure
	c := make(chan os.Signal, 0)
	var sd []chan bool
	for i := 0; i < 7; i++ {
		sd = append(sd, make(chan bool))
	}

	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func(c chan os.Signal, sd []chan bool) {
		<-c
		if !*us {
			fmt.Println("\nClosing YugenFlow Server")
			//var wg sync.WaitGroup
			for _, ch := range sd {
				//wg.Add(1)
				//go func(ch chan bool) {
				ch <- true
				//<-ch
				select {
				case <-ch:
				case <-time.After(time.Duration(globals.SettleTime) * time.Second):
				}
				//	wg.Done()
				//}(ch)
			}
			//wg.Wait()
			diskCache.Close()
			if err := coredbs.Disconnect(); err != nil {
				fmt.Println("Error in disconnecting from the YugenFlow Database")
			} else {
				fmt.Println("Disconnected from YugenFlow Database")
			}
			time.Sleep(time.Duration(globals.SettleTime) * time.Second)
		}
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
	if *dev && *stress == -1 {
		fmt.Println("*** WARNING: Development mode enabled ***")
		go sensormodels.SensorModel(0, 5000, 10, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x01}, false)
		go sensormodels.SensorModel(1, 5000, 10, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x02}, false)
		go sensormodels.SensorModel(2, 5000, 10, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x03}, false)
		go sensormodels.SensorModel(3, 5000, 10, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x04}, false)
		//go sensormodels.SensorModel(17, 50, 50, []int{-1, 1}, []byte{0x0a, 0x0b, 0x0c, 0x01, 0x02, 0x05})
	}

	//goland:noinspection ALL
	go gateManager.Start(sd[3])
	//goland:noinspection ALL
	go avgsManager.Start(sd[4])
	//goland:noinspection ALL
	go apiManager.Start(sd[5])

	fmt.Printf("\nYugenFlow Server active on ports %v , %v\n\n", globals.TCPport, globals.APIport)

	if *stress > 0 {
		fmt.Println("*** WARNING: Stress test mode enabled, make sure the correct configuration.ini is used ***")
		for id := 0; id < *stress; id++ {
			go sensormodels.SensorModel(id, 1, 10, []int{-1, 1}, []byte(fmt.Sprintf("%6d", id)), true)
			time.Sleep(50 * time.Millisecond)
		}
	}

	//goland:noinspection ALL
	exportManager.Start(sd[6])

}
