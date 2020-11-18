package apiManager

import (
	"encoding/json"
	"fmt"
	"gateserver/avgsManager"
	"gateserver/dataformats"
	"gateserver/entryManager"
	"gateserver/gateManager"
	"gateserver/spaceManager"
	"gateserver/storage/coredbs"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"github.com/gorilla/mux"
	"gopkg.in/ini.v1"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// returns the installation information
func info() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ApiManagerLog,
					mlogger.LoggerData{"apiManager.info",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()
		var installationInfo []JsonSpace

		// we need to set installationInfo every time to account for dunamic changes
		gateManager.GateStructure.RLock()
		entryManager.EntryStructure.RLock()
		spaceManager.SpaceStructure.RLock()

		//fmt.Println(gateManager.GateStructure.SensorList)
		//fmt.Println(entryManager.EntryStructure.GateList)
		//fmt.Println(spaceManager.SpaceStructure.EntryList)

		for spaceName, entryList := range spaceManager.SpaceStructure.EntryList {
			newSpace := JsonSpace{Id: spaceName}
			for entryName, entry := range entryList {
				newEntry := JsonEntry{
					Id:       entryName,
					Reversed: entry.Reversed,
				}
				for gateName, gate := range entryManager.EntryStructure.GateList[entryName] {
					newGate := JsonGate{
						Id:       gateName,
						Reversed: gate.Reversed,
					}
					for deviceName, device := range gateManager.GateStructure.SensorList[gateName] {
						newDevice := JsonDevices{
							Id:        deviceName,
							Reversed:  device.Reversed,
							Suspected: device.Suspected != 0,
							Disabled:  device.Disabled,
						}
						newGate.Devices = append(newGate.Devices, newDevice)
					}
					newEntry.Gates = append(newEntry.Gates, newGate)
				}
				newSpace.Entries = append(newSpace.Entries, newEntry)
			}
			installationInfo = append(installationInfo, newSpace)
		}

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		gateManager.GateStructure.RUnlock()
		entryManager.EntryStructure.RUnlock()
		spaceManager.SpaceStructure.RUnlock()

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(installationInfo)

	})
}

func connectedSensors() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ApiManagerLog,
					mlogger.LoggerData{"apiManager.connectedSensors",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()
		var connectedSensors []JsonConnectedDevice

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if macs, status, err := diskCache.ListActiveDevices(); err == nil {
			if len(macs) == len(status) {
				for i, mac := range macs {
					connectedSensors = append(connectedSensors,
						JsonConnectedDevice{
							Mac:    mac,
							Active: status[i],
						})
				}
			}
		}

		_ = json.NewEncoder(w).Encode(connectedSensors)

	})
}

func invalidSensors() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ApiManagerLog,
					mlogger.LoggerData{"apiManager.invalidSensors",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()
		var invalidSensorsList []JsonInvalidDevice

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if macs, timestamps, err := diskCache.ListInvalidDevices(); err == nil {
			if len(macs) == len(timestamps) {
				for i, mac := range macs {
					invalidSensorsList = append(invalidSensorsList,
						JsonInvalidDevice{
							Mac: mac,
							Ts:  timestamps[i],
						})
				}
			}
		}

		_ = json.NewEncoder(w).Encode(invalidSensorsList)

	})
}

func measurementDefinitions() http.Handler {
	// these definitions do not change over time
	var measurements []JsonMeasurement

	// load definitions of measurements from measurements.ini
	definitions, err := ini.InsensitiveLoad("measurements.ini")
	if err != nil {
		fmt.Printf("Fail to read measurements.ini file: %v\n", err)
		os.Exit(0)
	}

	for _, def := range definitions.Section("realtime").KeyStrings() {
		duration := definitions.Section("realtime").Key(def).MustInt(0)
		if duration != 0 {
			measurements = append(measurements,
				JsonMeasurement{
					Name:     def,
					Type:     "realtime",
					Interval: duration,
				})
		}
	}

	actualsAvailable := definitions.Section("system").Key("actuals").MustBool(false)
	if actualsAvailable {
		measurements = append(measurements,
			JsonMeasurement{
				Name:     "actual",
				Type:     "reference",
				Interval: 0,
			})
	}

	for _, def := range definitions.Section("reference").KeyStrings() {
		duration := definitions.Section("reference").Key(def).MustInt(0)
		if duration != 0 {
			measurements = append(measurements,
				JsonMeasurement{
					Name:     def,
					Type:     "reference",
					Interval: duration,
				})
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ApiManagerLog,
					mlogger.LoggerData{"apiManager.measurementDefinitions",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		_ = json.NewEncoder(w).Encode(measurements)

	})
}

func latestData(all, nonSeriesUseDB, seriesUseDB bool, which int) http.Handler {
	// which :
	// 0 - all
	// 1 - real time / delta
	// 2 - reference
	// 3 - presence

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ApiManagerLog,
					mlogger.LoggerData{"apiManager.latestData",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()

		if nonSeriesUseDB && seriesUseDB {
			mlogger.Error(globals.ApiManagerLog,
				mlogger.LoggerData{"apiManager.latestData",
					"internal error nonSeriesUseDB && seriesUseDB true",
					[]int{1}, true})
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(nil)
		}

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		var spaces []string
		spaceManager.SpaceStructure.RLock()

		if all {
			for name := range spaceManager.SpaceStructure.EntryList {
				spaces = append(spaces, name)
			}
		} else {
			vars := mux.Vars(r)
			if string(vars["space"][0]) == "[" && string(vars["space"][len(vars["space"])-1]) == "]" {
				list := strings.Split(vars["space"][1:len(vars["space"])-1], ",")
				for _, v := range list {
					name := strings.Trim(v, " ")
					if _, ok := spaceManager.SpaceStructure.EntryList[name]; ok {
						spaces = append(spaces, name)
					}
				}
			} else {
				if _, ok := spaceManager.SpaceStructure.EntryList[vars["space"]]; ok {
					spaces = []string{vars["space"]}
				}
			}
		}
		spaceManager.SpaceStructure.RUnlock()

		var results []JsonData
		if !nonSeriesUseDB && !seriesUseDB {
			if which == 0 || which == 1 {
				avgsManager.RegRealTimeChannels.RLock()
				for _, name := range spaces {
					select {
					case data := <-avgsManager.RegRealTimeChannels.ChannelOut[name]:
						newData := JsonData{
							Space:   name,
							Type:    "realtime",
							Results: make(map[string][]dataformats.MeasurementSample),
						}
						for measure, measurement := range data {
							newData.Results[measure] = []dataformats.MeasurementSample{measurement}
						}
						results = append(results, newData)
					default:
						// we do not wait for the channel to be ready
						//fmt.Println("data not ready, get skipped")
					}
				}
				avgsManager.RegRealTimeChannels.RUnlock()
			}

			if which == 0 || which == 2 {
				avgsManager.RegReferenceChannels.RLock()
				for _, name := range spaces {
					select {
					case data := <-avgsManager.RegReferenceChannels.ChannelOut[name]:
						newData := JsonData{
							Space:   name,
							Type:    "reference",
							Results: make(map[string][]dataformats.MeasurementSample),
						}
						for measure, measurement := range data {
							newData.Results[measure] = []dataformats.MeasurementSample{measurement}
						}
						results = append(results, newData)
					default:
						// we do not wait for the channel to be ready
						//fmt.Println("data not ready, get skipped")
					}
				}
				avgsManager.RegReferenceChannels.RUnlock()
			}

		} else if nonSeriesUseDB {
			numberSamples := 1
			options := strings.Split(r.URL.String(), "?")[1:]
			if len(options) <= 1 {
				if len(options) == 1 {
					if val, err := strconv.Atoi(options[0]); err == nil {
						numberSamples = val
					}
				}
				//}
				//for _, rp := range strings.Split(r.URL.String(), "?")[1:] {
				//	if val, err := strconv.Atoi(rp); err == nil {
				//		numberSamples = val
				//	}
				//}
				switch which {
				case 2:
					for _, name := range spaces {
						if data, err := coredbs.ReadReferenceData(name, numberSamples); err == nil {
							newData := JsonData{
								Space:   name,
								Type:    "reference",
								Results: make(map[string][]dataformats.MeasurementSample),
							}
							for _, val := range data {
								newData.Results[val.Qualifier] = append(newData.Results[val.Qualifier], val)
							}
							results = append(results, newData)
						}
					}
				case 1:
					for _, name := range spaces {
						if data, err := coredbs.ReadSpaceData(name, numberSamples); err == nil {
							newData := JsonData{
								Space:   name,
								Type:    "delta",
								Results: make(map[string][]dataformats.MeasurementSample),
							}
							for _, val := range data {
								newData.Results["delta"] = append(newData.Results["delta"], val)
							}
							results = append(results, newData)
						}
					}
				}
			}
		} else if seriesUseDB {
			options := strings.Split(r.URL.String(), "?")[1:]
			if len(options) == 2 {
				var t0, t1 int
				var err error
				if t0, err = strconv.Atoi(options[0]); err == nil {
					if t1, err = strconv.Atoi(options[1]); err == nil {
						if t0 != t1 {
							if t0 > t1 {
								tmp := t1
								t1 = t0
								t0 = tmp
							}
							//if which ==1 || which ==3 {
							//	t0 *= 1000000000
							//	t1 *= 1000000000
							//}
							//for _, name := range spaces {
							switch which {
							case 1:
								t0 *= 1000000000
								t1 *= 1000000000
								for _, name := range spaces {
									if data, err := coredbs.ReadSpaceDataSeries(name, t0, t1); err == nil {
										newData := JsonData{
											Space:   name,
											Type:    "delta",
											Results: make(map[string][]dataformats.MeasurementSample),
										}
										for _, val := range data {
											newData.Results["delta"] = append(newData.Results["delta"], val)
										}
										results = append(results, newData)
									}
								}
							case 2:
								for _, name := range spaces {
									if data, err := coredbs.ReadReferenceDataSeries(name, t0, t1); err == nil {
										newData := JsonData{
											Space:   name,
											Type:    "reference",
											Results: make(map[string][]dataformats.MeasurementSample),
										}
										for _, val := range data {
											newData.Results[val.Qualifier] = append(newData.Results[val.Qualifier], val)
										}
										results = append(results, newData)
									}
								}
							case 3:
								t0 *= 1000000000
								t1 *= 1000000000
								var answer []JsonPresence
								for _, name := range spaces {
									if presence, err := coredbs.VerifyPresence(name, t0, t1); err == nil {
										newData := JsonPresence{
											Space:    name,
											Presence: presence,
										}
										answer = append(answer, newData)
									}
								}
								w.WriteHeader(http.StatusOK)
								_ = json.NewEncoder(w).Encode(answer)
								return
							}
							//}
						}
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(results)

	})
}

func command() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ApiManagerLog,
					mlogger.LoggerData{"apiManager.info",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		vars := mux.Vars(r)

		var answer JsonCmdRt

		params := make(map[string]string)
		params["cmd"] = vars["command"]

		if options := strings.Split(r.URL.String(), "?"); len(options) != 0 {
			for _, val := range options[1:] {
				if option := strings.Split(val, "="); len(option) == 2 {
					params[strings.ToLower(strings.Trim(option[0], " "))] = strings.ToLower(strings.Trim(option[1], " "))
				} else {
					answer.Error = "error parameter " + val
					break
				}
			}
			if async, ok := params["async"]; ok && async == "1" {
				go executeCommand(params)
				answer.Answer = "ok"
			} else {
				answer = executeCommand(params)
			}
		} else {
			answer.Error = "syntax error"
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(answer)

	})
}
