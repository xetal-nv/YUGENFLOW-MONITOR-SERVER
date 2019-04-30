package sensormodels

import (
	"encoding/binary"
	"fmt"
	"gateserver/codings"
	"math/rand"
	"net"
	"os"
	"strings"
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

// for testing purposes only

func SensorModel(id, iter, mxdelay int, vals []int, mac []byte) {

	del := rand.Intn(mxdelay) + 1
	time.Sleep(2 * time.Second)
	mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", []byte(mac)), " ", ":", -1), ":")
	fmt.Printf("FakeSensor %v // %v -> Connect to TCP channel\n", id, mach)
	port := os.Getenv("TCPPORT")
	conn, e := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	//noinspection GoUnhandledErrorResult
	if e != nil || conn == nil {
		fmt.Println("Unable to connect")
	} else {
		defer conn.Close()
		// sensor registers first
		//noinspection GoUnhandledErrorResult
		conn.Write(mac)
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
						fmt.Printf("Sensor %v has received data %v\n", mach, cmd)
						select {
						case c <- cmd:
						case <-time.After(5 * time.Second):
							fmt.Printf("sensor %v timeout\n", mach)
						}
					}
				}
			}
		}(c)
		for i := 0; i < iter; i++ {
			// sensor model loop starts with seding a data element
			rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
			data := vals[rand.Intn(len(vals))]
			bs := make([]byte, 4)
			binary.BigEndian.PutUint32(bs, uint32(id))
			//noinspection GoUnhandledErrorResult
			msg := []byte{1, bs[2], bs[3], byte(data)}
			msg = append(msg, codings.Crc8(msg))
			//fmt.Println(id, msg)
			if _, e = conn.Write(msg); e == nil {
				// fork for either sending a new data value or receiving a command
				select {
				case v := <-c:
					fmt.Printf("sensor %v command accepted\n", mach)
					crc := codings.Crc8(v[:len(v)-1])
					if crc == v[len(v)-1] {
						msg := []byte{v[0]}
						if rt, ok := command[v[0]]; ok {
							msg = append(msg, rt...)
						}
						crc = codings.Crc8(msg)
						msg = append(msg, crc)
						fmt.Printf("Sensor %v answering command %v\n", mach, msg)
						_, e = conn.Write(msg)
						if v[0] == 14 {
							fmt.Printf("Sensor %v disconnecting with new id\n", mach)
							go func() {
								time.Sleep(10 * time.Second)
								id = int(v[2]) + int(v[1])*256
								SensorModel(id, iter-i, mxdelay, vals, mac)
							}()
							return
						}
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

	fmt.Printf("FakeSensor %v // %v disconnecting\n", id, mach)
}
