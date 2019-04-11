package servers

import (
	"context"
	"gateserver/support"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// set-up of the TCP server
func setUpTCP() {
	if os.Getenv("CRC") == "1" {
		crcUsed = true
	} else {
		crcUsed = false
	}

	if v, e := strconv.Atoi(os.Getenv("DEVICETO")); e != nil {
		timeout = 5
	} else {
		timeout = v
	}

	if v, e := strconv.Atoi(os.Getenv("MALTO")); e != nil {
		maltimeout = 600
	} else {
		maltimeout = v
	}

	resetbg.start, resetbg.end, resetbg.interval, resetbg.valid = time.Time{}, time.Time{}, time.Duration(0), false
	rng := strings.Split(strings.Trim(os.Getenv("RESETSLOT"), ";"), ";")
	if len(rng) == 3 {
		if v, e := time.Parse(support.TimeLayout, strings.Trim(rng[0], " ")); e == nil {
			resetbg.start = v
			if v, e = time.Parse(support.TimeLayout, strings.Trim(rng[1], " ")); e == nil {
				resetbg.end = v
				if v, e := strconv.Atoi(strings.Trim(rng[2], " ")); e == nil {
					if v != 0 {
						resetbg.interval = time.Duration(v)
						resetbg.valid = true
					}
				}
			}
		}
	}

	log.Println("servers.StartTCP: CRC usage is set to", crcUsed)

	mutexUnknownMac.Lock()
	mutexSensorMacs.Lock()
	sensorChanID = make(map[int]chan []byte)
	sensorChanUsedID = make(map[int]bool)
	SensorCmdID = make(map[int]chan []byte)
	sensorMacID = make(map[int][]byte)
	sensorIdMAC = make(map[string]int)
	unknownMacChan = make(map[string]chan []byte)
	unkownDevice = make(map[string]bool)
	unusedDevice = make(map[int]string)
	mutexSensorMacs.Unlock()
	mutexUnknownMac.Unlock()
}

// start of the TCP server, including set-up
func StartTCP(sd chan context.Context) {

	setUpTCP()

	// Listen for incoming connections.
	port := os.Getenv("TCPPORT")
	l, e := net.Listen(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	if e != nil {
		log.Fatal("servers.StartTCP: fatal error:", e)
	}

	r := func() {
		<-sd
		//noinspection GoUnhandledErrorResult
		l.Close()
	}

	go support.RunWithRecovery(r, nil)

	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"servers.StartTCP: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			log.Println("servers.StartTCP: recovering server", port, "from:\n", e)
			sd <- context.Background() // close running shutdown goroutine
			//noinspection GoUnhandledErrorResult
			l.Close()
			StartTCP(sd)
		}
	}()

	log.Printf("servers.StartTCP: listening on 0.0.0.0:%v\n", port)
	for {
		// Listen for an incoming connection.
		conn, e := l.Accept()
		if e != nil {
			log.Printf("servers.StartTCP: Error accepting: %v\n", e)
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handlerTCPRequest(conn)
	}
}
