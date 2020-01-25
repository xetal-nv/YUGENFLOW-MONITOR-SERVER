package servers

import (
	"fmt"
	"gateserver/gates"
	"gateserver/support"
	"log"
	"net"
	"strconv"
	"time"
)

// handles the periodical background reset, when enabled
func handlerReset(id int) {
	if id < 0 {
		go func() {
			support.DLog <- support.DevData{"servers.handlerReset device " + strconv.Itoa(id),
				support.Timestamp(), "illegal id", []int{1}, false}
		}()
		return
	}
	log.Printf("servers.handlerReset: reset enabled for Device %v\n from %v to %v every %v/n", id, resetbg.start, resetbg.end, resetbg.interval*time.Minute)
	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"servers.handlerReset: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			handlerReset(id)
		}
	}()
	done := false
	for {
		time.Sleep(resetbg.interval * time.Minute)
		//fmt.Println("resetting device", id)
		if doit, e := support.InClosureTime(resetbg.start, resetbg.end); e == nil {
			if doit {
				if !done {
					rt := exeBinaryCommand(strconv.Itoa(id), "rstbg", []int{})
					if rt.State {
						//fmt.Println(rt.State)
						done = true
						go func() {
							support.DLog <- support.DevData{"servers.handlerReset: reset device " + strconv.Itoa(id),
								support.Timestamp(), "", []int{1}, true}
						}()
						// releases possible request on rstReq
						// missing a reset request is impossible since the reset just happened
						gates.SensorRst.RLock()
						if resetChannel, ok := gates.SensorRst.Channel[id]; ok {
							go func(req chan bool) {
								select {
								case <-req:
									fmt.Println("emptied reset channel", id)
								case <-time.After(500 * time.Millisecond):
								}
							}(resetChannel)
						}
						gates.SensorRst.RUnlock()
					} else {
						go func() {
							support.DLog <- support.DevData{"servers.handlerReset: failed to reset device " + strconv.Itoa(id),
								support.Timestamp(), "", []int{1}, true}
						}()
					}
				}
			} else {
				done = false
				// check if there is a reset request pending
				//fmt.Println("checking pending reset request for", id)
				gates.SensorRst.RLock()
				resetChannel, ok := gates.SensorRst.Channel[id]
				gates.SensorRst.RUnlock()
				if ok {
					select {
					case <-resetChannel:
						// TODO serve request for debug
						//fmt.Println("resetting device", id)
						//noinspection GoUnusedCallResult
						exeBinaryCommand(strconv.Itoa(id), "rstbg", []int{})
					case <-time.After(500 * time.Millisecond):
					}
				}
			}
		} else {
			log.Printf("servers.handlerReset: device %v has reset error %v\n", id, e)
		}
	}
}

func assingID(st chan bool, conn net.Conn, com chan net.Conn, _mac []byte) {
	defer func() { st <- false }()
	select {
	case <-com:
		com <- conn
		<-com
	case <-time.After(time.Duration(maltimeout) * time.Second):
	}
}
