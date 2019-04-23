package main

import (
	"gateserver/gates"
	"gateserver/sensormodels"
	"gateserver/servers"
	"gateserver/spaces"
	"gateserver/storage"
	"gateserver/support"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {

	folder, _ := support.GetCurrentExecDir()

	folder = os.Getenv("GATESERVER")

	var e error
	if folder != "" {
		e = os.Chdir(folder)
	}

	cleanup := func() {
		log.Println("System shutting down")
		support.SupportTerminate()
		storage.TimedIntDBSClose()
	}
	support.SupportSetUp("")

	if folder != "" {
		if e == nil {
			log.Printf("Move to folder %v\n", folder)
		} else {
			log.Fatal("Unable to move to folder %v, error reported:%v\n", folder, e)
		}
	}

	// Set-up databases
	if err := storage.TimedIntDBSSetUp(false); err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	gates.SetUp()
	spaces.SetUp()

	// testing
	switch os.Getenv("DEVMODE") {
	case "2":
		for i := 100; i < 200; i++ {
			mac := []byte{'a', 'b', 'c'}
			mac = append(mac, []byte(strconv.Itoa(i))...)
			go sensormodels.SensorModel(i-100, 100, 10, []int{-1, 0, 1, 2, 127}, mac)
		}
		for i := 200; i < 300; i++ {
			mac := []byte{'a', 'b', 'c'}
			mac = append(mac, []byte(strconv.Itoa(i))...)
			go sensormodels.SensorModel(65535, 100, 10, []int{-1, 0, 1, 2, 127}, mac)
		}
	case "1":
		//go sensormodels.Randgen()
		go sensormodels.SensorModel(0, 110, 2, []int{-1, 0, 1, 2, 127}, []byte{'a', 'b', 'c', '1', '2', '1'})
		go sensormodels.SensorModel(1, 120, 3, []int{-1, 0, 1, 2, 127}, []byte{'a', 'b', 'c', '1', '2', '2'})
		go sensormodels.SensorModel(2, 50, 2, []int{-1, 0, 1, 2, 127}, []byte{'a', 'b', 'c', '1', '2', '3'})
		go sensormodels.SensorModel(3, 70, 3, []int{-1, 0, 1, 2, 127}, []byte{'a', 'b', 'c', '1', '2', '4'})
		go sensormodels.SensorModel(7, 90, 2, []int{-1, 0, 1, 2, 127}, []byte{'a', 'b', 'c', '1', '2', '5'})
		go sensormodels.SensorModel(65535, 50, 2, []int{-1, 0, 1, 2, 127}, []byte{'a', 'b', 'c', '1', '2', '6'})
		//go func(){
		//	sensormodels.SensorModel(3, 1, 2, []int{-1, 0, 1, 2, 127},[]byte{'a', 'b', 'c', 1, 2, 4})
		//	time.Sleep(10*time.Second)
		//	sensormodels.SensorModel(3, 1, 2, []int{-1, 0, 1, 2, 127},[]byte{'a', 'b', 'c', 1, 2, 3})
		//}()
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
