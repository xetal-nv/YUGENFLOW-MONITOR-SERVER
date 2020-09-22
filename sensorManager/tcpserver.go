package sensorManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"log"
	"net"
	"time"
	"xetal.ddns.net/utils/recovery"
)

// TODO add sensor number limit with tokens
func tcpServer(rst chan bool) {
	srv, e := net.Listen("tcp4", "0.0.0.0:"+globals.TCPport)
	if e != nil {
		log.Fatal("sensorManager.tcpServer: fatal error:", e)
	}
	defer srv.Close()

	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.tcpServer",
			"listening on 0.0.0.0:" + globals.TCPport,
			[]int{1}, true})
	for {
		c := make(chan net.Conn, 1)
		go func(c chan net.Conn, srv net.Listener) {
			conn, e := srv.Accept()
			if e == nil {
				select {
				case c <- conn:
					return
				case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
				}
			} else {
				if globals.DebugActive {
					log.Printf("sensorManager.tcpServer: Error accepting: %v\n", e)
				}
			}
		}(c, srv)
		select {
		case nc := <-c:
			if globals.DebugActive {
				log.Printf("sensorManager.tcpServer: device connected\n")
			}
			go recovery.CleanPanic(
				func() { handler(nc) },
				func() {
					//goland:noinspection GoUnhandledErrorResult
					nc.Close()
				})
		case <-rst:
			fmt.Println("Closing sensorManager.tcpServer")

			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.tcpServer",
					"service stopped",
					[]int{1}, true})
			rst <- true

		}
	}
}
