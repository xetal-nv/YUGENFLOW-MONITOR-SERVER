package sensorManager

import (
	"fmt"
	codings "gateserver/codings"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"sync"
	"time"
)

func sensorCommand(chs SensorChannel, mac string) {
	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
			"service started",
			[]int{0}, true})
	if globals.DebugActive {
		fmt.Println("sensorManager.sensorCommand: started", mac)
	}
	allowedUnexpectedAnswers := globals.FailureThreshold
finished:
	for {
		// we return always nil when there are no errors, something otherwise
		select {
		case <-time.After(time.Duration(globals.ZombieTimeout) * time.Hour):
			// we check if this routine is a zombie and terminate it in that case
			ActiveSensors.RLock()
			newChs, active := ActiveSensors.Mac[mac]
			ActiveSensors.RUnlock()
			if !active {
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
						"service killed due to zombie timeout",
						[]int{0}, true})
				// closing the channel causes the TCP handler also to close
				if chs.Tcp != nil {
					_ = chs.Tcp.Close()
				}
				return
			} else if newChs.Tcp != chs.Tcp || chs.Tcp == nil {
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
						"service killed due to zombie timeout",
						[]int{0}, true})
				// closing the channel causes the TCP handler also to close
				if chs.Tcp != nil {
					_ = chs.Tcp.Close()
				}
				return
			}
		case <-chs.reset:
			// reset request received, the routine is terminated normally
			break finished
		case ans := <-chs.CmdAnswer:
			if allowedUnexpectedAnswers == 0 {
				if globals.DebugActive {
					fmt.Println("sensorManager.sensorCommand:", mac, "too many unexpected data, connection closed")
				}
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
						"too many unexpected data, connection closed",
						[]int{1}, true})
				// closing the channel causes the TCP handler also to close
				_ = chs.Tcp.Close()
				return
			}
			// this is an unsolicited command answer, we report error
			if globals.DebugActive {
				fmt.Printf("!!! unexpected answer %x fpr %v !!!\n", ans, mac)
			}
			select {
			case chs.CmdAnswer <- []byte("e"):
			case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
				// if this happens we force a TCP responder reset
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
						"service killed due to sensor timeout",
						[]int{0}, true})
				//_ = chs.Tcp.Close()
				return
			case <-chs.reset:
				break finished
			}
			allowedUnexpectedAnswers -= 1
		case cmd := <-chs.Commands:
			// this is a command request
			// we return nil to the command issuer in case of failed and not null to the commander
			// responder in case of error
			var rtIssuer []byte
			rtResponder := []byte("e")
			// verify if the command exists and send it to the device
			if _, ok := CmdAnswerLen[cmd[0]]; ok {
				if globals.DebugActive {
					fmt.Printf("Received command %v for device %v\n", cmd, mac)
					if cmd[0] == CmdAPI["setid"].Cmd {
						fmt.Printf("Changing id to %v for device %v\n", int(cmd[2]), mac)
					}
				}
				cmd = append(cmd, codings.Crc8(cmd))
				ready := make(chan bool)
				go func(ba []byte) {
					ret := false
					if e := chs.Tcp.SetWriteDeadline(time.Now().Add(time.Duration(globals.SensorTimeout) * time.Second)); e == nil {
						if _, e := chs.Tcp.Write(ba); e == nil {
							ret = true
						}
					}
					select {
					case ready <- ret:
					case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
					}
				}(cmd)
				select {
				case valid := <-ready:
					if valid {
						select {
						case rtIssuer = <-chs.CmdAnswer:
							// we check if the answer is semantically correct
							if rtIssuer[0] == cmd[0] {
								rtResponder = nil
							}
						case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
						case <-chs.reset:
							break finished
						}
						// make optional
						if cmd[0] == CmdAPI["rstbg"].Cmd && globals.ResetCloseTCP {
							//println("bye")
							_ = chs.Tcp.Close()
						}
					}
				case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
				case <-chs.reset:
					break finished
				}
			}
			// in case of timeout, the receiving party will timeout and the TCP responder will force a reset
			var wg sync.WaitGroup
			wg.Add(2)

			fn := func(ch chan dataformats.Commanding, msg dataformats.Commanding) {
				defer wg.Done()
				select {
				case ch <- msg:
				case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
				}
			}
			go fn(chs.Commands, rtIssuer)
			go fn(chs.CmdAnswer, rtResponder)
			wg.Wait()
			allowedUnexpectedAnswers = globals.FailureThreshold
		}
	}
	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
			"service stopped",
			[]int{0}, true})
	if globals.DebugActive {
		fmt.Println("sensorManager.sensorCommand: ended", mac)
	}
	select {
	case <-time.After(time.Duration(globals.ZombieTimeout) * time.Hour):
	case chs.reset <- true:
	}

}
