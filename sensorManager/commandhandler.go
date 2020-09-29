package sensorManager

import (
	"fmt"
	"gateserver/codings"
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
finished:
	for {
		// we return always nil when there are no errors, something otherwise
		select {
		case <-time.After(time.Duration(globals.ZombieTimeout) * time.Hour):
			// we assume this routine is a zombie and terminate it
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
					"service killed due to zombie timeout",
					[]int{0}, true})
			return
		case <-chs.reset:
			// reset request received, the routine is terminated normally
			break finished
		case ans := <-chs.CmdAnswer:
			// this is an unsolicited command answer, we report error
			if globals.DebugActive {
				fmt.Printf("unexpected answer %x fpr %v\n", ans, mac)
			}
			select {
			case chs.CmdAnswer <- []byte("e"):
			case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
				// if this happens we force a TCP responder reset
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
						"service killed due to sensor timeout",
						[]int{0}, true})
				//_ = chs.tcp.Close()
				return
			case <-chs.reset:
				break finished
			}
		case cmd := <-chs.Commands:
			// this is a command request
			// we return nil to the command issuer in case of failed and not null to the commander responder in case of error
			var rtIssuer []byte
			rtResponder := []byte("e")
			// verify if the command exists and send it to the device
			if _, ok := cmdAnswerLen[cmd[0]]; ok {
				if globals.DebugActive {
					fmt.Printf("Received command %v for device %v\n", cmd, mac)
					if cmd[0] == cmdAPI["setid"].cmd {
						fmt.Printf("Changing id to %v for device %v\n", int(cmd[2]), mac)
					}
				}
				cmd = append(cmd, codings.Crc8(cmd))
				ready := make(chan bool)
				go func(ba []byte) {
					ret := false
					if e := chs.tcp.SetWriteDeadline(time.Now().Add(time.Duration(globals.SensorTimeout) * time.Second)); e == nil {
						if _, e := chs.tcp.Write(ba); e == nil {
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
					}
				case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
				case <-chs.reset:
					break finished
				}
			}
			// in case of timeout, the receiving party will timeout and the TCP responder will force a reset
			var wg sync.WaitGroup
			wg.Add(2)

			fn := func(ch chan dataformats.Commandding, msg dataformats.Commandding) {
				defer wg.Done()
				select {
				case ch <- msg:
				case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
				}
			}
			go fn(chs.Commands, rtIssuer)
			go fn(chs.CmdAnswer, rtResponder)
			wg.Wait()
		}
	}
	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.sensorCommand: " + mac,
			"service stopped",
			[]int{0}, true})
	if globals.DebugActive {
		fmt.Println("sensorManager.sensorCommand: ended", mac)
	}
	chs.reset <- true
}
