package main

import (
	"flag"
	"gateserver/gates"
	"gateserver/sensormodels"
	"gateserver/servers"
	"gateserver/spaces"
	"gateserver/storage"
	"gateserver/support"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {

	folder, _ := support.GetCurrentExecDir()

	// TODO add support foi command line options

	var env = flag.String("env", "", "configuration filename")
	flag.Parse()

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
	support.SupportSetUp(*env)

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
		log.Println("WARNING DEVELOPMENT MODE 2")
		for i := 100; i < 300; i++ {
			mac := []byte{'a', 'b', 'c'}
			mac = append(mac, []byte(strconv.Itoa(i))...)
			go func(i int, mac []byte) {
				time.Sleep(time.Duration(rand.Intn(360)) * time.Second)
				sensormodels.SensorModel(i-100, 500, 60, []int{-1, 0, 1, 2, 127}, mac)
			}(i, mac)
		}
		for i := 300; i < 450; i++ {
			mac := []byte{'a', 'b', 'c'}
			mac = append(mac, []byte(strconv.Itoa(i))...)
			go func(i int, mac []byte) {
				time.Sleep(time.Duration(rand.Intn(360)) * time.Second)
				sensormodels.SensorModel(65535, 500, 60, []int{-1, 0, 1, 2, 127}, mac)
			}(i, mac)
		}
	case "1":
		log.Println("WARNING DEVELOPMENT MODE 1")
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
