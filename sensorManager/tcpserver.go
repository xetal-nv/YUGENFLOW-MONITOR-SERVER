package sensorManager

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"net"
	"os"
	"time"
	"xetal.ddns.net/utils/recovery"
)

func tcpServer(rst chan interface{}) {

	srv, e := net.Listen("tcp4", "0.0.0.0:"+globals.TCPport)
	if e != nil {
		fmt.Println("sensorManager.tcpServer: fatal error:", e)
		os.Exit(0)
	}
	defer srv.Close()

	tokens = make(chan interface{}, MAXTCP)
	for i := MAXTCP; i > 0; i-- {
		tokens <- nil
	}

	if globals.DebugActive {
		fmt.Printf("*** WARNING: maximum number of concurrent TCP connections is limited to %v ***\n", len(tokens))
	}

	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.tcpServer",
			"listening on 0.0.0.0:" + globals.TCPport,
			[]int{0}, true})
	for {
		c := make(chan net.Conn, 1)
		go func(c chan net.Conn, srv net.Listener) {
			// accept and if there are no token wait and close
		validconnection:
			for {
				conn, e := srv.Accept()
				if e == nil {
					select {
					case <-tokens:
						if globals.DebugActive {
							fmt.Printf("sensorManager.tcpServer: acquired token, left: %v\n", len(tokens))
						}
						select {
						case c <- conn:
							break validconnection
						case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
							if globals.DebugActive {
								fmt.Printf("sensorManager.tcpServer: released token, left: %v\n", len(tokens))
							}
							_ = conn.Close()
							tokens <- nil
						}
					case <-time.After((time.Duration(globals.SensorTimeout) * time.Second)):
						_ = conn.Close()
						mlogger.Info(globals.SensorManagerLog,
							mlogger.LoggerData{"sensorManager.tcpServer token service",
								"no token left",
								[]int{1}, true})
					}
				} else {
					if globals.DebugActive {
						fmt.Printf("sensorManager.tcpServer: Error accepting: %v\n", e)
					}
				}
			}
		}(c, srv)
		select {
		case nc := <-c:
			if globals.DebugActive {
				fmt.Printf("sensorManager.tcpServer: device connected\n")
			}
			go recovery.CleanPanic(
				func() { handler(nc) },
				func() {
					// this is redundant
					//goland:noinspection GoUnhandledErrorResult
					nc.Close()
				})
		case <-rst:
			fmt.Println("Closing sensorManager.tcpServer")
			// we stop all running sensor processes
			ActiveSensors.Lock()
			for _, el := range ActiveSensors.Mac {
				_ = el.tcp.Close()
				el.reset <- true
				<-el.reset
			}
			ActiveSensors.Unlock()
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.tcpServer",
					"service stopped",
					[]int{0}, true})
			rst <- nil

		}
	}
}
