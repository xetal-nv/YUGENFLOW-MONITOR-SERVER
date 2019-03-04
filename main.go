package main

import (
	"countingserver/gates"
	"countingserver/servers"
	"countingserver/spaces"
	"countingserver/storage"
	"countingserver/support"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	defer support.SupportTerminate()
	support.SupportSetUp("")

	// Set-up databases
	if err := storage.TimedIntDBSSetUp(false); err != nil {
		log.Fatal(err)
	}
	defer storage.TimedIntDBSClose()

	gates.SetUp()
	spaces.SetUp()

	// testing
	if os.Getenv("DEVMODE") != "" {
		go fake_devices()
	}

	// comment below for TCP debug
	// Set-up and start servers
	servers.StartServers()

	// Uncomment below for TCP debug
	//gates.SetUp()
	//spaces.SetUp()
	//servers.StartTCP(make(chan context.Context))

	// for the API use the globals SpaceDef and EntryList to extract the entire installation logical structure

}

func fake_devices() {
	iter := 20
	vals := []int{-1, 0, 1, 2, 127}
	devices := []int{0, 1}

	time.Sleep(2 * time.Second)
	fmt.Println(" TEST -> Connect to TCP channel")
	port := os.Getenv("TCPPORT")
	conn, e := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	if e != nil {
		fmt.Println("Unable to connect")
	} else {
		//noinspection GoUnhandledErrorResult
		conn.Write([]byte{'a', 'b', 'c', 1, 2, 3})
		for i := 0; i < iter; i++ {
			rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
			data := vals[rand.Intn(len(vals))]
			dev := devices[rand.Intn(len(devices))]
			//noinspection GoUnhandledErrorResult
			conn.Write([]byte{1, 0, byte(dev), byte(data)})
			time.Sleep(1000 * time.Millisecond)

		}
	}
	//noinspection GoUnhandledErrorResult
	conn.Close()
	fmt.Println(" TEST -> Disconnect to TCP channel")
}
