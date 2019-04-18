package servers

import (
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
	log.Printf("servers.handlerReset: reset enabled for Device %v\n", id)
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
		if doit, e := support.InClosureTime(resetbg.start, resetbg.end); e == nil {
			if doit {
				if !done {
					rt := exeBinaryCommand(strconv.Itoa(id), "rstbg", []int{})
					if rt.State {
						done = true
						go func() {
							support.DLog <- support.DevData{"servers.handlerReset: reset device " + strconv.Itoa(id),
								support.Timestamp(), "", []int{1}, true}
						}()
					} else {
						go func() {
							support.DLog <- support.DevData{"servers.handlerReset: failed to reset device " + strconv.Itoa(id),
								support.Timestamp(), "", []int{1}, true}
						}()
					}
				}
			} else {
				done = false
			}
		} else {
			log.Printf("servers.handlerReset: device %v has reset error %v\n", id, e)
		}
	}
}

func assingID(st chan bool, conn net.Conn, com chan net.Conn, mac []byte) {
	defer func() { st <- false }()
	//fmt.Println("start command routine")
	select {
	case <-com:
		com <- conn
		<-com
	case <-time.After(time.Duration(maltimeout) * time.Second):
	}
}
