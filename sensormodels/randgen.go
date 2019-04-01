package sensormodels

import (
	"fmt"
	"gateserver/codings"
	"math/rand"
	"net"
	"os"
	"time"
)

// for testing purposes only

func _Randgen() {
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
		//conn.Write([]byte{2})
		for i := 0; i < iter; i++ {
			rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
			data := vals[rand.Intn(len(vals))]
			dev := devices[rand.Intn(len(devices))]
			//fmt.Println("sending", data)
			//noinspection GoUnhandledErrorResult
			msg := []byte{1, 0, byte(dev), byte(data)}
			msg = append(msg, codings.Crc8(msg))
			if _, e := conn.Write(msg); e != nil {
				break
			}
			time.Sleep(1000 * time.Millisecond)
			if i == 5 {
				msg := []byte{2}
				msg = append(msg, codings.Crc8(msg))
				fmt.Println("sending something else", msg)
				_, _ = conn.Write(msg)
				time.Sleep(1000 * time.Millisecond)
			}

		}
	}
	//noinspection GoUnhandledErrorResult
	if conn != nil {
		conn.Close()
	}
	fmt.Println(" TEST -> Disconnect to TCP channel")
}
