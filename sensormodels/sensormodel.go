package sensormodels

import (
	"countingserver/codings"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

var command = map[byte][]byte{
	[]byte("\x07")[0]: []byte("\x01\x02"),
	[]byte("\x09")[0]: []byte("\x01\x02"),
	[]byte("\x0b")[0]: []byte("\x01\x02"),
	[]byte("\x0d")[0]: []byte("\x01\x02"),
}

// TODO to be verified
func SensorModel(id, iter, mxdelay int, vals []int) {

	time.Sleep(2 * time.Second)
	fmt.Println(" TEST -> Connect to TCP channel")
	port := os.Getenv("TCPPORT")
	conn, e := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	//noinspection GoUnhandledErrorResult
	defer conn.Close()
	if e != nil {
		fmt.Println("Unable to connect")
	} else {
		// sensor registers first
		//noinspection GoUnhandledErrorResult
		conn.Write([]byte{'a', 'b', 'c', 1, 2, 3})
		// start a listener
		c := make(chan []byte, 5)
		go func(c chan []byte) {
			var e error
			for e == nil {
				cmd := make([]byte, 1)
				if _, e = conn.Read(cmd); e == nil {
					fmt.Printf("Sensor %v has received data %v\n", id, cmd)
					c <- cmd
				}
			}
		}(c)
		for i := 0; i < iter; i++ {
			// sensor model loop starts with seding a data element
			rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
			data := vals[rand.Intn(len(vals))]
			del := rand.Intn(mxdelay) + 1
			//noinspection GoUnhandledErrorResult
			msg := []byte{1, 0, byte(id), byte(data)}
			msg = append(msg, codings.Crc8(msg))
			_, _ = conn.Write(msg)
			// fork for either sending a new data value or receiving a command
			select {
			case v := <-c:
				crc := codings.Crc8(v[:len(v)-1])
				if crc == v[len(v)-1] {
					msg := []byte{v[0]}
					if rt, ok := command[v[0]]; ok {
						msg = append(msg, rt...)
					}
					crc = codings.Crc8(msg)
					msg = append(msg, crc)
					_, e = conn.Write(msg)
				}
			case <-time.After(time.Duration(del) * 1000 * time.Millisecond):
				// continue to send data
			}
		}
	}

	fmt.Printf("Sensor %v disconnecting\n", id)
}
