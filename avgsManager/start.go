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
		//var wg sync.WaitGroup
		for _, ch := range rstC {
			//wg.Add(1)
			//go func(ch chan interface{}) {
			ch <- nil
			select {
			case <-ch:
			case <-time.After(time.Duration(globals.SettleTime) * time.Second):
			}
			//wg.Done()
			//}(ch)
		}
		//wg.Wait()
		mlogger.Info(globals.AvgsManagerLog,
			mlogger.LoggerData{"avgsManager.Start",
				"service stopped",
				[]int{0}, true})
		time.Sleep(time.Duration(globals.SettleTime) * time.Second)
		sd <- true
	}(sd, rstC)

	var maxTick int = 0
	realTimeDefinitions := make(map[string]int)
	referenceDefinitions := make(map[string]int)

	// load definitions of measurements from measurements.ini
	definitions, err := ini.InsensitiveLoad("measurements.ini")
	if err != nil {
		fmt.Printf("Fail to read measurements.ini file: %v\n", err)
		os.Exit(0)
	}

	tick := definitions.Section("system").Key("tick").MustInt(5)
	actualsAvailable := definitions.Section("system").Key("actuals").MustBool(false)

	for _, def := range definitions.Section("realtime").KeyStrings() {
		duration := definitions.Section("realtime").Key(def).MustInt(0)
		if duration != 0 {
			realTimeDefinitions[def] = duration
			if duration > maxTick {
				maxTick = duration
			}
		} else {
			fmt.Printf("Measurement definition for %v is invalid\n", def)
		}
	}

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
	LatestData.Channel = make(map[string]chan dataformats.SpaceState)
	RegRealTimeChannels.channelIn = make(map[string]chan dataformats.SimpleSample)
	RegRealTimeChannels.ChannelOut = make(map[string]chan map[string]dataformats.SimpleSample)
	RegReferenceChannels.channelIn = make(map[string]chan dataformats.SimpleSample)
	RegReferenceChannels.ChannelOut = make(map[string]chan map[string]dataformats.SimpleSample)

	for i := 0; i < len(listSpaces)-1; i++ {
		name := listSpaces[i]
		ldChan := make(chan dataformats.SpaceState, globals.ChannellingLength)
		LatestData.Channel[name] = ldChan
		regRTIn := make(chan dataformats.SimpleSample, globals.ChannellingLength)
		regRTOut := make(chan map[string]dataformats.SimpleSample, globals.ChannellingLength)
		RegRealTimeChannels.channelIn[name] = regRTIn
		RegRealTimeChannels.ChannelOut[name] = regRTOut
		regRfIn := make(chan dataformats.SimpleSample, globals.ChannellingLength)
		regRfOut := make(chan map[string]dataformats.SimpleSample, globals.ChannellingLength)
		RegReferenceChannels.channelIn[name] = regRfIn
		RegReferenceChannels.ChannelOut[name] = regRfOut
		go recovery.RunWith(
			func() {
				calculator(name, ldChan, rstC[i], tick, maxTick, realTimeDefinitions, referenceDefinitions,
					regRTIn, regRfIn, actualsAvailable)
			},
			func() {
				mlogger.Recovered(globals.AvgsManagerLog,
					mlogger.LoggerData{"avgsManager.calculator for space: " + listSpaces[i],
						"service terminated and recovered unexpectedly",
						[]int{1}, true})
			})
		go LatestMeasurementRegister(name+"_realtime", regRTIn, regRTOut, nil)

		go LatestMeasurementRegister(name+"_reference", regRfIn, regRfOut, nil)

	}
	last := len(listSpaces) - 1
	name := listSpaces[last]
	LldChan := make(chan dataformats.SpaceState, globals.ChannellingLength)
	LatestData.Channel[name] = LldChan
	regRTIn := make(chan dataformats.SimpleSample, globals.ChannellingLength)
	regRTOut := make(chan map[string]dataformats.SimpleSample, globals.ChannellingLength)
	RegRealTimeChannels.channelIn[name] = regRTIn
	RegRealTimeChannels.ChannelOut[name] = regRTOut
	regRfIn := make(chan dataformats.SimpleSample, globals.ChannellingLength)
	regRfOut := make(chan map[string]dataformats.SimpleSample, globals.ChannellingLength)
	RegReferenceChannels.channelIn[name] = regRfIn
	RegReferenceChannels.ChannelOut[name] = regRfOut

	RegRealTimeChannels.Unlock()
	LatestData.Unlock()

	go LatestMeasurementRegister(name+"_realtime", regRTIn, regRTOut, nil)

	go LatestMeasurementRegister(name+"_reference", regRfIn, regRfOut, nil)

	recovery.RunWith(
		func() {
			calculator(name, LldChan, rstC[last], tick, maxTick, realTimeDefinitions, referenceDefinitions,
				regRTIn, regRfIn, actualsAvailable)
		},
		func() {
			mlogger.Recovered(globals.AvgsManagerLog,
				mlogger.LoggerData{"avgsManager.calculator for space: " + listSpaces[len(listSpaces)-1],
					"service terminated and recovered unexpectedly",
					[]int{1}, true})
		})

	//for {
	//	time.Sleep(36 * time.Hour)
	//}
}
