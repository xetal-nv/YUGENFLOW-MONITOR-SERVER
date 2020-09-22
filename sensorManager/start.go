package sensorManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/dbs/sensorDB"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"xetal.ddns.net/utils/recovery"
)

func Start(sd chan bool) {
	var err error

	if globals.SensorManagerLog, err = mlogger.DeclareLog("yugenflow_sensorManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_sensorManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.SensorManagerLog, 40, 30, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.SensorManagerLog,
		mlogger.LoggerData{"sensorManager.Start",
			"service started",
			[]int{1}, true})

	var rstC []chan bool
	for i := 0; i < 1; i++ {
		rstC = append(rstC, make(chan bool))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan bool) {
		<-sd
		fmt.Println("Closing sensorManager")
		var wg sync.WaitGroup
		for _, ch := range rstC {
			wg.Add(1)
			go func(ch chan bool) {
				ch <- true
				select {
				case <-ch:
				case <-time.After(2 * time.Second):
				}
				wg.Done()
			}(ch)
		}
		wg.Wait()
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensorManager.Start",
				"service stopped",
				[]int{1}, true})
		time.Sleep(time.Duration(globals.ShutdownTime) * time.Second)
		sd <- true
	}(sd, rstC)

	// First we load any eventual sensor declaration
	ActiveSensors.Mac = make(map[string]SensorChannel)
	ActiveSensors.Id = make(map[int]string)
	foundDefault := false

	for _, mac := range globals.Config.Section("sensors").KeyStrings() {
		if sensorDeclarationRaw := globals.Config.Section("sensors").Key(mac).Value(); sensorDeclarationRaw != "" {
			sensorDeclaration := strings.Split(sensorDeclarationRaw, " ")
			for i, el := range sensorDeclaration {
				sensorDeclaration[i] = strings.Trim(el, " ")
			}
			if id, err := strconv.Atoi(sensorDeclaration[0]); err == nil {
				fn := func(a string, list []string) bool {
					for _, b := range list {
						if b == a {
							return true
						}
					}
					return false
				}

				if err := sensorDB.AddLookUp([]byte(strconv.Itoa(id)), mac); err == nil {
					if err = sensorDB.WriteDefinition([]byte(mac), dataformats.SensorDefinition{
						Id:      id,
						Bypass:  fn("bypass", sensorDeclaration[1:]),
						Report:  fn("report", sensorDeclaration[1:]),
						Enforce: fn("enforce", sensorDeclaration[1:]),
						Strict:  fn("strict", sensorDeclaration[1:]),
					}); err != nil {
						_ = sensorDB.DeleteLookUp([]byte(sensorDeclaration[0]))
						mlogger.Panic(globals.SensorManagerLog,
							mlogger.LoggerData{"sensorManager.Start",
								"failed to load declaration for " + mac, []int{}, false}, true)
						time.Sleep(time.Duration(globals.ShutdownTime) * time.Second)
						os.Exit(0)
					}
					foundDefault = (mac == "default")
				} else {
					mlogger.Panic(globals.SensorManagerLog,
						mlogger.LoggerData{"sensorManager.Start",
							"failed to load declaration for " + mac, []int{}, false}, true)
					time.Sleep(time.Duration(globals.ShutdownTime) * time.Second)
					os.Exit(0)
				}
			}
		}
	}
	if !foundDefault {
		mlogger.Panic(globals.SensorManagerLog,
			mlogger.LoggerData{"sensorManager.Start",
				"default sensor declaration missing", []int{1}, true}, true)
		time.Sleep(time.Duration(globals.ShutdownTime) * time.Second)
		os.Exit(0)
	}

	recovery.RunWith(
		func() { tcpServer(rstC[0]) },
		func() {
			mlogger.Recovered(globals.SensorManagerLog,
				mlogger.LoggerData{"sensorManager.tcpServer",
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		})

	//for {
	//	time.Sleep(36 * time.Hour)
	//}
}
