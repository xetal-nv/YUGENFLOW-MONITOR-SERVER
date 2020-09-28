package sensorManager

import (
	"fmt"
	"gateserver/supp"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"strconv"
	"strings"
	"time"
)

func sensorBGReset(forceReset chan string, rst chan bool) {

	resetFn := func(channels SensorChannel) bool {
		// TODO to be done
		println("reset BG")
		return true
	}

	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.sensorBGReset",
			"service started",
			[]int{0}, true})
	if globals.ResetSlot != "" {
		var start, stop time.Time
		period := strings.Split(globals.ResetSlot, " ")
		valid := false
		if v, e := time.Parse(supp.TimeLayout, strings.Trim(period[0], " ")); e == nil {
			start = v
			if v, e = time.Parse(supp.TimeLayout, strings.Trim(period[1], " ")); e == nil {
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
					rst <- true
					return
				case mac := <-forceReset:
					ActiveSensors.RLock()
					chs, ok := ActiveSensors.Mac[mac]
					ActiveSensors.RUnlock()
					if ok {
						// reset
						if resetFn(chs) {
							mac = ""
						}
					}
					forceReset <- mac
				case <-time.After(time.Duration(globals.ResetPeriod) * time.Minute):
					if doIt, e := supp.InClosureTime(start, stop); e == nil {
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
			rst <- true
			return
		case mac := <-forceReset:
			ActiveSensors.RLock()
			chs, ok := ActiveSensors.Mac[mac]
			ActiveSensors.RUnlock()
			if ok {
				// reset
				if resetFn(chs) {
					mac = ""
				}
			}
			forceReset <- mac
		}
	}
}
