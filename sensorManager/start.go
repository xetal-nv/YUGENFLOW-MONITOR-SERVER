package sensorManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"strconv"
	"strings"
	"time"
	"xetal.ddns.net/utils/recovery"
)

func Start(sd chan bool) {
	var err error

	if globals.SensorManagerLog, err = mlogger.DeclareLog("yugenflow_sensorManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_sensorManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.SensorManagerLog, 50, 50, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.Start",
			"service started",
			[]int{0}, true})

	var rstC []chan interface{}
	for i := 0; i < 2; i++ {
		rstC = append(rstC, make(chan interface{}))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan interface{}) {
		<-sd
		fmt.Println("Closing sensorManager")
		for _, ch := range rstC {
			ch <- nil
			select {
			case <-ch:
			case <-time.After(time.Duration(globals.SettleTime) * time.Second):
			}
		}
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensorManager.Start",
				"service stopped",
				[]int{0}, true})
		time.Sleep(time.Duration(globals.SettleTime) * time.Second)
		sd <- true
	}(sd, rstC)

	// First we load any eventual sensor declaration
	ActiveSensors.Mac = make(map[string]SensorChannel)
	ActiveSensors.Id = make(map[int]string)

	for _, mac := range globals.Config.Section("sensors").KeyStrings() {
		if sensorDeclarationRaw := globals.Config.Section("sensors").Key(mac).Value(); sensorDeclarationRaw != "" {
			sensorDeclaration := strings.Split(sensorDeclarationRaw, " ")
			for i, el := range sensorDeclaration {
				sensorDeclaration[i] = strings.Trim(el, " ")
			}
			if id, err := strconv.Atoi(sensorDeclaration[0]); err == nil {
				if id < 0 {
					mlogger.Warning(globals.SensorManagerLog,
						mlogger.LoggerData{"sensorManager.Start",
							"illegal declaration for mac " + mac,
							[]int{0}, false})
					continue
				}
				fn := func(a string, list []string) bool {
					for _, b := range list {
						if b == a {
							return true
						}
					}
					return false
				}

				//if err := diskCache.AddLookUp([]byte(strconv.Itoa(id)), mac); err == nil {
				if len(sensorDeclaration) > 2 {
					definition := dataformats.SensorDefinition{
						Id:      id,
						Bypass:  fn("bypass", sensorDeclaration[2:]),
						Report:  fn("report", sensorDeclaration[2:]),
						Enforce: fn("enforce", sensorDeclaration[2:]),
						Strict:  fn("strict", sensorDeclaration[2:]),
					}
					// bypass has priority on strict
					definition.Strict = definition.Strict && !definition.Bypass
					// enforce does nothing if strict is given
					definition.Enforce = definition.Enforce && !definition.Strict

					if rate, err := strconv.Atoi(sensorDeclaration[1]); err == nil {
						definition.MaxRate = int64(rate) * 1000000
					} else {
						mlogger.Panic(globals.SensorManagerLog,
							mlogger.LoggerData{"sensorManager.Start",
								"failed to load declaration for " + mac, []int{0}, false}, true)
						time.Sleep(time.Duration(globals.SettleTime) * time.Second)
						os.Exit(0)
					}

					if err = diskCache.WriteDefinition([]byte(mac), definition); err != nil {
						_ = diskCache.DeleteLookUp([]byte(sensorDeclaration[0]))
						mlogger.Panic(globals.SensorManagerLog,
							mlogger.LoggerData{"sensorManager.Start",
								"failed to load declaration for " + mac, []int{0}, false}, true)
						time.Sleep(time.Duration(globals.SettleTime) * time.Second)
						os.Exit(0)
					}
				} else {
					if globals.DebugActive {
						fmt.Println("illegal sensor declaration given for mac", mac)
					}
					mlogger.Warning(globals.SensorManagerLog,
						mlogger.LoggerData{"sensorManager.Start",
							"illegal declaration for mac " + mac,
							[]int{0}, false})
				}
			}
		}
	}

	go recovery.RunWith(
		func() { tcpServer(rstC[0]) },
		func() {
			mlogger.Recovered(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.tcpServer",
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		})

	recovery.RunWith(
		func() { sensorBGReset(globals.ResetChannel, rstC[1]) },
		func() {
			mlogger.Recovered(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.sensorBGReset",
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		})

}
