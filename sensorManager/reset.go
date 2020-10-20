package sensorManager

import (
	"fmt"
	"gateserver/codings"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"strconv"
	"strings"
	"time"
)

func sensorBGReset(forceReset chan string, rst chan interface{}) {

	// return true is successful
	resetFn := func(channels SensorChannel) bool {
		cmd := []byte{CmdAPI["rstbg"].Cmd}
		cmd = append(cmd, codings.Crc8(cmd))
		var res []byte
		select {
		case channels.Commands <- cmd:
			select {
			case res = <-channels.Commands:
			case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
			case <-rst:
				fmt.Println("Closing sensorManager.sensorBGReset")
				mlogger.Info(globals.SensorManagerLog,
					mlogger.LoggerData{"sensorManager.sensorBGReset",
						"service stopped",
						[]int{0}, true})
				rst <- nil
			}
		case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
		}
		return res != nil
	}

	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.sensorBGReset",
			"service started",
			[]int{0}, true})
	if globals.ResetSlot != "" {
		var start, stop time.Time
		period := strings.Split(globals.ResetSlot, " ")
		valid := false
		if v, e := time.Parse(globals.TimeLayout, strings.Trim(period[0], " ")); e == nil {
			start = v
			if v, e = time.Parse(globals.TimeLayout, strings.Trim(period[1], " ")); e == nil {
				stop = v
				valid = true
			}
		}
		if valid {
			if globals.DebugActive {
				fmt.Printf("*** INFO: sensor reset is set from %v tp %v\n", start, stop)
			}
			var channels []SensorChannel
			var macs []string
			done := false
			for {
				select {
				case <-rst:
					fmt.Println("Closing sensorManager.sensorBGReset")
					mlogger.Info(globals.SensorManagerLog,
						mlogger.LoggerData{"sensorManager.sensorBGReset",
							"service stopped",
							[]int{0}, true})
					rst <- nil
					return
				case mac := <-forceReset:
					// an answer is not needed back to the reset commander
					ActiveSensors.RLock()
					chs, ok := ActiveSensors.Mac[mac]
					ActiveSensors.RUnlock()
					if ok {
						// reset
						if !resetFn(chs) {
							mlogger.Warning(globals.SensorManagerLog,
								mlogger.LoggerData{"sensorManager.sensorBGReset: mac " + mac,
									"reset has failed",
									[]int{1}, true})
						}
					} else {
						mlogger.Warning(globals.SensorManagerLog,
							mlogger.LoggerData{"sensorManager.sensorBGReset: mac " + mac,
								"reset skipped as sensor not active",
								[]int{0}, true})
					}
				case <-time.After(time.Duration(globals.ResetPeriod) * time.Minute):
					if doIt, e := others.InClosureTime(start, stop); e == nil {
						//  then cycle among all devices till all are reset
						if doIt && !done {
							// we are in the reset interval and we still need to reset
							if channels == nil {
								// in this case we need to load the list of devices to be reset
								ActiveSensors.RLock()
								for k, v := range ActiveSensors.Mac {
									macs = append(macs, k)
									channels = append(channels, v)
								}
								ActiveSensors.RUnlock()
							}
							var channelsLeft []SensorChannel
							var macsLeft []string
							// try to reset all devices
							for i, el := range channels {
								if !resetFn(el) {
									channelsLeft = append(channelsLeft, el)
									macsLeft = append(macsLeft, macs[i])
								} else {
									if globals.DebugActive {
										fmt.Println("sensorManager.sensorBGReset:", macs[i],
											"BG reset executed")
									}
									mlogger.Info(globals.SensorManagerLog,
										mlogger.LoggerData{"sensorManager.sensorBGReset: " + macs[i],
											"BG reset executed",
											[]int{0}, true})
								}
							}
							if channelsLeft != nil {
								copy(channels, channelsLeft)
								copy(macs, macsLeft)
							} else {
								channels = nil
								macs = nil
							}

							if channelsLeft == nil {
								//println("done BGreset")
								done = true
							}
						} else {
							if !doIt {
								done = false
								if channels != nil {
									mlogger.Warning(globals.SensorManagerLog,
										mlogger.LoggerData{"sensorManager.sensorBGReset",
											"service failed for " + strconv.Itoa(len(channels)) + " sensors",
											[]int{}, false})
									channels = nil
								}
							}
						}
					} else {
						// error
						mlogger.Error(globals.SensorManagerLog,
							mlogger.LoggerData{"sensorManager.sensorBGReset",
								"service failed to initialised new loop",
								[]int{1}, true})
					}
				}
			}
		}
	}
	if globals.DebugActive {
		fmt.Println("*** WARNING: periodic sensor reset is disabled ***")
	}
	// we only listed to reset and forceReset
	for {
		select {
		case <-rst:
			fmt.Println("Closing sensorManager.sensorBGReset")
			mlogger.Info(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.sensorBGReset",
					"service stopped",
					[]int{0}, true})
			rst <- nil
			return
		case mac := <-forceReset:
			// an answer is not needed back to the reset commander
			ActiveSensors.RLock()
			chs, ok := ActiveSensors.Mac[mac]
			ActiveSensors.RUnlock()
			if ok {
				// reset
				if !resetFn(chs) {
					mlogger.Warning(globals.SensorManagerLog,
						mlogger.LoggerData{"sensorManager.sensorBGReset: mac " + mac,
							"reset has failed",
							[]int{1}, true})
				}
			} else {
				mlogger.Warning(globals.SensorManagerLog,
					mlogger.LoggerData{"sensorManager.sensorBGReset: mac " + mac,
						"reset skipped as sensor not active",
						[]int{0}, true})
			}
			//select {
			//case forceReset <- mac:
			//case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
			//	// we detach the answe with a zombie timeout
			//	go func() {
			//		select {
			//		case forceReset <- mac:
			//		case <-time.After(time.Duration(globals.ZombieTimeout) * time.Hour):
			//			mlogger.Warning(globals.SensorManagerLog,
			//				mlogger.LoggerData{"sensorManager.sensorBGReset",
			//					"potential zombie",
			//					[]int{1}, true})
			//		}
			//	}()
			//}
		}
	}
}
