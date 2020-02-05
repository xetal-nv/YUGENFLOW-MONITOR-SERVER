package servers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"gateserver/codings"
	"gateserver/gates"
	"gateserver/support"
	"log"
	"math"
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
						go func() {
							support.DLog <- support.DevData{"servers.handlerReset: reset due to sensor asymmetry " + strconv.Itoa(id),
								support.Timestamp(), "", []int{1}, true}
						}()
						//fmt.Println("resetting device", id)
						_ = exeBinaryCommand(strconv.Itoa(id), "rstbg", []int{})
					case <-time.After(500 * time.Millisecond):
					}
				}

			}
		} else {
			log.Printf("servers.handlerReset: device %v has reset error %v\n", id, e)
		}
	}
}

//noinspection GoUnusedParameter
func assingID(st chan bool, conn net.Conn, com chan net.Conn, _mac []byte) {
	defer func() { st <- false }()
	select {
	case <-com:
		com <- conn
		<-com
	case <-time.After(time.Duration(maltimeout) * time.Second):
	}
}

// setSensorParameters sets the sensor parameter TBD (try few times before making error)
// TODO to be tested on the field
func setSensorParameters(conn net.Conn, mac string) (err error) {

	// sends the command for a maximum of eepromResetTries times before reporting error
	sendCommand := func(command string, value uint32) (e error) {
		e = errors.New("server.setSensorParameters: Failed to execute command " + command)
		timeout := 1
		//mainLoop:
		for i := 0; i < eepromResetTries; i++ {
			if v, ok := cmdAPI[command]; ok {
				cmd := []byte{v.cmd}
				bs := make([]byte, 4)
				//binary.BigEndian.PutUint32(bs, uint32(specs.srate))
				binary.BigEndian.PutUint32(bs, value)
				cmd = append(cmd, bs[4-v.lgt:4]...)
				cmd = append(cmd, codings.Crc8(cmd))
				if e := conn.SetWriteDeadline(time.Now().Add(time.Duration(timeout) * time.Second)); e == nil {
					if _, e := conn.Write(cmd); e == nil {
						// loop on read till we find a significant answer
					readLoop:
						for {
							ans := make([]byte, 1)
							if e := conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second)); e == nil {
								if _, e := conn.Read(ans); e != nil {
									switch ans[0] {
									case 1:
										// sensor data, discard the rest of the message
										var data []byte
										if crcUsed {
											data = make([]byte, 4)
										} else {
											data = make([]byte, 3)
										}
										_, _ = conn.Read(data)
									case cmd[0]:
										// answer to command, discard CRC if rpesent
										if crcUsed {
											_, _ = conn.Read(make([]byte, 1))
										}
										//break mainLoop
										return nil
									default:
										// illegal answer
										break readLoop
									}
								}
							}
						}
					}
				}
			}
		}
		return
	}

	if !SensorEEPROMResetEnables {
		return
	} else {
		if specs, ok := sensorData[mac]; ok {

			eLab := "("
			if e := sendCommand("srate", uint32(specs.srate)); e != nil {
				log.Println(e)
				eLab += "srate "
			}
			if e := sendCommand("savg", uint32(specs.savg)); e != nil {
				log.Println(e)
				eLab += "savg "
			}
			if e := sendCommand("bgth", uint32(math.Round(specs.bgth*16))); e != nil {
				log.Println(e)
				eLab += "bgth "
			}
			if e := sendCommand("occth", uint32(math.Round(specs.occth*16))); e != nil {
				log.Println(e)
				eLab += "occth "
			}
			if eLab != "" {
				err = errors.New("Failed to execute commands " + eLab + ") for device " + mac)
			}
		} else {
			go func() {
				support.DLog <- support.DevData{"servers.SensorEEPROMResetEnables: sensorData cache is corrupted",
					support.Timestamp(), "", []int{1}, true}
			}()
			err = errors.New("servers.SensorEEPROMResetEnables: sensorData cache is corrupted for device " + mac)
		}
		return
	}
}
