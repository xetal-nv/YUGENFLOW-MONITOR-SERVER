package avgsManager

import (
	"fmt"
	"gateserver/dataformats"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"gopkg.in/ini.v1"
	"os"
	"time"
	"xetal.ddns.net/utils/recovery"
)

func Start(sd chan bool) {

	var err error

	if globals.AvgsManagerLog, err = mlogger.DeclareLog("yugenflow_avgsManager", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_avgsManager logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(globals.AvgsManagerLog, 50, 50, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(globals.AvgsManagerLog,
		mlogger.LoggerData{"avgsManager.Start",
			"service started",
			[]int{0}, true})

	var listSpaces []string
	for _, sp := range globals.Config.Section("spaces").KeyStrings() {
		listSpaces = append(listSpaces, sp)
	}

	if len(listSpaces) == 0 {
		fmt.Printf("No spaces are defined in configuration.ini file\n")
		os.Exit(0)
	}

	var rstC []chan interface{}
	for i := 0; i < len(listSpaces); i++ {
		rstC = append(rstC, make(chan interface{}))
	}

	// setting up closure and shutdown
	go func(sd chan bool, rstC []chan interface{}) {
		<-sd
		fmt.Println("Closing avgsManager")
		for _, ch := range rstC {
			ch <- nil
			select {
			case <-ch:
			case <-time.After(time.Duration(globals.SettleTime) * time.Second):
			}
		}
		mlogger.Info(globals.AvgsManagerLog,
			mlogger.LoggerData{"avgsManager.Start",
				"service stopped",
				[]int{0}, true})
		time.Sleep(time.Duration(globals.SettleTime) * time.Second)
		sd <- true
	}(sd, rstC)

	var maxTick = 0
	var tick = 15
	realTimeDefinitions := make(map[string]int)
	referenceDefinitions := make(map[string]int)

	// load definitions of measurements from measurements.ini
	definitions, err := ini.InsensitiveLoad(globals.WorkPath + "measurements.ini")
	if err != nil {
		fmt.Printf("Fail to read measurements.ini file: %v\n", err)
		os.Exit(0)
	}

	currentAvailable := definitions.Section("system").Key("current").MustBool(false)

	for _, def := range definitions.Section("realtime").KeyStrings() {
		duration := definitions.Section("realtime").Key(def).MustInt(0)
		if duration != 0 {
			realTimeDefinitions[def] = duration
			if duration > maxTick {
				maxTick = duration
			}
			if duration < tick {
				tick = duration
			}
		} else {
			fmt.Printf("Measurement definition for %v is invalid\n", def)
		}
	}

	tick = int(tick / 3)

	for _, def := range definitions.Section("reference").KeyStrings() {
		duration := definitions.Section("reference").Key(def).MustInt(0)
		if duration != 0 {
			referenceDefinitions[def] = duration
			if duration > maxTick {
				maxTick = duration
			}
		} else {
			fmt.Printf("Measurement definition for %v is invalid\n", def)
		}
	}

	LatestData.Lock()
	RegRealTimeChannels.Lock()
	RegReferenceChannels.Lock()
	RegActualChannels.Lock()
	LatestData.Channel = make(map[string]chan dataformats.SpaceState)
	RegRealTimeChannels.channelIn = make(map[string]chan dataformats.MeasurementSample)
	RegRealTimeChannels.ChannelOut = make(map[string]chan map[string]dataformats.MeasurementSample)
	RegReferenceChannels.channelIn = make(map[string]chan dataformats.MeasurementSample)
	RegReferenceChannels.ChannelOut = make(map[string]chan map[string]dataformats.MeasurementSample)
	RegActualChannels.channelIn = make(map[string]chan dataformats.MeasurementSampleWithFlows)
	RegActualChannels.ChannelOut = make(map[string]chan dataformats.MeasurementSampleWithFlows)

	for i := 0; i < len(listSpaces)-1; i++ {
		name := listSpaces[i]
		ldChan := make(chan dataformats.SpaceState)
		LatestData.Channel[name] = ldChan
		regRTIn := make(chan dataformats.MeasurementSample)
		regRTOut := make(chan map[string]dataformats.MeasurementSample)
		RegRealTimeChannels.channelIn[name] = regRTIn
		RegRealTimeChannels.ChannelOut[name] = regRTOut
		regRfIn := make(chan dataformats.MeasurementSample)
		regRfOut := make(chan map[string]dataformats.MeasurementSample)
		RegReferenceChannels.channelIn[name] = regRfIn
		RegReferenceChannels.ChannelOut[name] = regRfOut
		regAcIn := make(chan dataformats.MeasurementSampleWithFlows)
		regAcOut := make(chan dataformats.MeasurementSampleWithFlows)
		RegActualChannels.channelIn[name] = regAcIn
		RegActualChannels.ChannelOut[name] = regAcOut
		go recovery.RunWith(
			func() {
				calculator(name, ldChan, rstC[i], tick, maxTick, realTimeDefinitions, referenceDefinitions, regRTIn, regRfIn, regAcIn, currentAvailable)
			},
			func() {
				mlogger.Recovered(globals.AvgsManagerLog,
					mlogger.LoggerData{"avgsManager.calculator for space: " + listSpaces[i],
						"service terminated and recovered unexpectedly",
						[]int{1}, true})
			})
		go LatestMeasurementRegister(name+"_realtime", regRTIn, regRTOut, nil)
		go LatestMeasurementRegister(name+"_reference", regRfIn, regRfOut, nil)
		go LatestMeasurementRegisterActuals(name+"_current", regAcIn, regAcOut)

	}
	// the last process cannot be a go routine
	last := len(listSpaces) - 1
	name := listSpaces[last]
	LldChan := make(chan dataformats.SpaceState)
	LatestData.Channel[name] = LldChan
	regRTIn := make(chan dataformats.MeasurementSample)
	regRTOut := make(chan map[string]dataformats.MeasurementSample)
	RegRealTimeChannels.channelIn[name] = regRTIn
	RegRealTimeChannels.ChannelOut[name] = regRTOut
	regRfIn := make(chan dataformats.MeasurementSample)
	regRfOut := make(chan map[string]dataformats.MeasurementSample)
	RegReferenceChannels.channelIn[name] = regRfIn
	RegReferenceChannels.ChannelOut[name] = regRfOut
	regAcIn := make(chan dataformats.MeasurementSampleWithFlows)
	regAcOut := make(chan dataformats.MeasurementSampleWithFlows)
	RegActualChannels.channelIn[name] = regAcIn
	RegActualChannels.ChannelOut[name] = regAcOut
	RegActualChannels.Unlock()
	RegReferenceChannels.Unlock()
	RegRealTimeChannels.Unlock()
	LatestData.Unlock()

	go LatestMeasurementRegister(name+"_realtime", regRTIn, regRTOut, nil)
	go LatestMeasurementRegister(name+"_reference", regRfIn, regRfOut, nil)
	go LatestMeasurementRegisterActuals(name+"_current", regAcIn, regAcOut)

	recovery.RunWith(
		func() {
			calculator(name, LldChan, rstC[last], tick, maxTick, realTimeDefinitions, referenceDefinitions, regRTIn, regRfIn, regAcIn, currentAvailable)
		},
		func() {
			//println("died")
			mlogger.Recovered(globals.AvgsManagerLog,
				mlogger.LoggerData{"avgsManager.calculator for space: " + listSpaces[len(listSpaces)-1],
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		})

}
