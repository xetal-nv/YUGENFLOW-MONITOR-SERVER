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
	[]byte("\x07")[0]: []byte("\x07\x07"),
	[]byte("\x09")[0]: []byte("\x09\x09"),
	[]byte("\x0b")[0]: []byte("\x0b\x0b"),
	[]byte("\x0d")[0]: []byte("\x0d\x0d"),
}

var cmdargs = map[byte]int{
	[]byte("\x02")[0]: 1,
	[]byte("\x03")[0]: 1,
	[]byte("\x04")[0]: 2,
	[]byte("\x05")[0]: 2,
	[]byte("\x0e")[0]: 2,
}

// TODO to be verified
func SensorModel(id, iter, mxdelay int, vals []int) {

	del := rand.Intn(mxdelay) + 1
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
		data := vals[rand.Intn(len(vals))]
		msg := []byte{1, 0, byte(id), byte(data)}
		msg = append(msg, codings.Crc8(msg))
		//noinspection GoUnhandledErrorResult
		conn.Write(msg)
		// start a listener
		c := make(chan []byte)
		go func(c chan []byte) {
			var e error
			for e == nil {
				cmd := make([]byte, 1)
				ll := 1
				if _, e = conn.Read(cmd); e == nil {
					if l, ok := cmdargs[cmd[0]]; ok {
						ll += l
					}
					cmde := make([]byte, ll)
					if _, e = conn.Read(cmde); e == nil {
						cmd = append(cmd, cmde...)
					}
					if e == nil {
						fmt.Printf("Sensor %v has received data %v\n", id, cmd)
						select {
						case c <- cmd:
						case <-time.After(5 * time.Second):
							fmt.Printf("sensor %v timeout\n", id)
						}
					}
				}
			}
		}(c)
		for i := 0; i < iter; i++ {
			// sensor model loop starts with seding a data element
			rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
			data = vals[rand.Intn(len(vals))]
			//noinspection GoUnhandledErrorResult
			msg = []byte{1, 0, byte(id), byte(data)}
			msg = append(msg, codings.Crc8(msg))
			if _, e = conn.Write(msg); e == nil {
				// fork for either sending a new data value or receiving a command
				select {
				case v := <-c:
					fmt.Printf("sensor %v command accepted\n", id)
					crc := codings.Crc8(v[:len(v)-1])
					if crc == v[len(v)-1] {
						msg := []byte{v[0]}
						if rt, ok := command[v[0]]; ok {
							msg = append(msg, rt...)
						}
						crc = codings.Crc8(msg)
						msg = append(msg, crc)
						fmt.Printf("Sensor %v answering command %v\n", id, msg)
						_, e = conn.Write(msg)
					}
				case <-time.After(time.Duration(del+5) * 1000 * time.Millisecond):
					// continue to send data
				}
			}
			if e != nil {
				break
			}
		}
	}

	fmt.Printf("Sensor %v disconnecting\n", id)
}
