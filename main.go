package main

import (
	"countingserver/gates"
	"countingserver/sensormodels"
	"countingserver/servers"
	"countingserver/spaces"
	"countingserver/storage"
	"countingserver/support"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

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
		cleanup()
		os.Exit(1)
	}()

	// comment below for TCP debug
	// Set-up and start servers
	servers.StartServers()
}
