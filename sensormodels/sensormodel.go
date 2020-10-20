package sensormodels

import (
	"encoding/binary"
	"fmt"
	"gateserver/codings"
	"gateserver/sensorManager"
	"gateserver/support/globals"
	"math/rand"
	"net"
	"strings"
	"time"
)

// gate sensor model for testing purposes only

func SensorModel(id, iter, mxdelay int, vals []int, mac []byte) {

	del := rand.Intn(mxdelay) + 1
	time.Sleep(2 * time.Second)
	mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", ":", -1), ":")
	fmt.Printf("FakeSensor %v // %v -> Connect to TCP channel\n", id, mach)
	conn, e := net.Dial("tcp4", "0.0.0.0:"+globals.TCPport)
	//noinspection GoUnhandledErrorResult
	if e != nil || conn == nil {
		fmt.Println("Unable to connect")
	} else {
		defer conn.Close()
		// sensor registers first
		//noinspection GoUnhandledErrorResult
		conn.Write(mac)
		// test pending
		// start a listener

		// test wrong malicious error
		//conn.Close()
		//conn, e = net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
		//if e != nil || conn == nil {
		//	fmt.Println("Unable to connect")
		//} else {
		//conn.Write(mac)

		// test end

		c := make(chan []byte)
		go func(c chan []byte) {
			var e error
			for e == nil {
				cmd := make([]byte, 1)
				ll := 1
				if _, e = conn.Read(cmd); e == nil {
					if l, ok := sensorManager.CmdAnswerLen[cmd[0]]; ok {
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
						case <-time.After(2 * time.Second):
							fmt.Printf("sensor %v timeout\n", mach)
						}
					}
				}
			}
		}(c)
		//time.Sleep(30 * time.Second)
		for i := 0; i < iter; i++ {
			// sensor model loop starts with sending a data element
			rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
			data := vals[rand.Intn(len(vals))]
			bs := make([]byte, 4)
			binary.BigEndian.PutUint32(bs, uint32(id))
			//noinspection GoUnhandledErrorResult
			msg := []byte{1, bs[2], bs[3], byte(data)}
			msg = append(msg, codings.Crc8(msg))
			//if _, e = conn.Write(msg); e == nil {
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
					//msg = append(msg, []byte("/")...)
					fmt.Printf("Sensor %v answering %v to received command\n", mach, msg)
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
				} else {
					fmt.Printf("sensor %v command wrong CRC\n", mach)
				}
			case <-time.After(time.Duration(del+5) * 1000 * time.Millisecond):
				// continue to send data
				//fmt.Println(string(mac), msg)
				_, e = conn.Write(msg)
			}
			//}
			if e != nil {
				fmt.Printf("FakeSensor %v // %v no longer connected to the server\n", id, mach)
				break
			}
		}
	}
	//}

	fmt.Printf("FakeSensor %v // %v disconnecting\n", id, mach)
}
