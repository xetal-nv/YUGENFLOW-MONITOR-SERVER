package main

import (
	"fmt"
	"gateserver/gates"
	"gateserver/sensormodels"
	"gateserver/servers"
	"gateserver/spaces"
	"gateserver/storage"
	"gateserver/support"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	folder, _ := support.GetCurrentExecDir()

	folder = os.Getenv("GATESERVER")
	fmt.Println(folder)

	if folder != "" {
		e := os.Chdir(folder)
		log.Printf("Move to folder %v\n", folder)
		if e != nil {
			log.Fatal("Unable to move to folder %v, error reported:%v\n", folder, e)
		}
	}

	cleanup := func() {
		log.Println("System shutting down")
		support.SupportTerminate()
		storage.TimedIntDBSClose()
	}
	support.SupportSetUp("")

	// Set-up databases
	if err := storage.TimedIntDBSSetUp(false); err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	gates.SetUp()
	spaces.SetUp()

	// testing
	switch os.Getenv("DEVMODE") {
	case "1":
		//go sensormodels.Randgen()
		go sensormodels.SensorModel(0, 100, 20, []int{-1, 0, 1, 2, 127})
		//go sensormodels.SensorModel(1, 100, 20, []int{-1, 0, 1, 2, 127})
		//go sensormodels.SensorModel(2, 100, 20, []int{-1, 0, 1, 2, 127})
		//go sensormodels.SensorModel(3, 100, 20, []int{-1, 0, 1, 2, 127})
	default:
	}

	// Capture all killing s
	c := make(chan os.Signal)
	//signal.Notify(c)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL,
		syscall.SIGQUIT, syscall.SIGABRT)
	go func() {
		<-c
		support.CleanupLock.Lock()
		cleanup()
		support.CleanupLock.Unlock()
		os.Exit(1)
	}()

	// comment below for TCP debug
	// Set-up and start servers
	servers.StartServers()
}
