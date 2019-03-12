package sensormodels

import (
	"countingserver/codings"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

// TODO make full sensor model with comands as well
func SensorModel(id, iter, mxdel int, vals []int) {

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
			del := rand.Intn(mxdel) + 1
			//noinspection GoUnhandledErrorResult
			msg := []byte{1, 0, byte(id), byte(data)}
			msg = append(msg, codings.Crc8(msg))
			_, _ = conn.Write(msg)
			time.Sleep(time.Duration(del) * 1000 * time.Millisecond)
			if i == 5 {
				msg := []byte{2}
				msg = append(msg, codings.Crc8(msg))
				fmt.Println("sending something else", msg)
				_, _ = conn.Write(msg)
				time.Sleep(time.Duration(del) * 1000 * time.Millisecond)
			}

		}
	}
	//noinspection GoUnhandledErrorResult
	conn.Close()
	fmt.Println(" TEST -> Disconnect to TCP channel")
}
