package apiManager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gateserver/avgsManager"
	"gateserver/dataformats"
	"gateserver/entryManager"
	"gateserver/gateManager"
	"gateserver/sensorManager"
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
	"time"
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

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(installationInfo)
			return
		}

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
		var connSensors []JsonConnectedDevice

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(connectedSensors)
			return
		}

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if macIdTable, err := diskCache.GenerateIdLookUp(); err == nil {
			if macs, status, err := diskCache.ListActiveDevices(); err == nil {
				if len(macs) == len(status) {
					for i, mac := range macs {
						id, ok := macIdTable[mac]
						if !ok {
							id = -1
						}
						connSensors = append(connSensors,
							JsonConnectedDevice{
								Mac:    mac,
								Id:     id,
								Active: status[i],
							})
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(connSensors)

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

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(invalidSensorsList)
			return
		}

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

		w.WriteHeader(http.StatusOK)
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

	currentAvailable := definitions.Section("system").Key("current").MustBool(false)
	if currentAvailable {
		measurements = append(measurements,
			JsonMeasurement{
				Name:     "current",
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

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(measurements)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(measurements)

	})
}

func latestData(all, nonSeriesUseDB, seriesUseDB bool, which int) http.Handler {
	// which :
	// 0 - all (includes current if enabled)
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

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(spaces)
			return
		}

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

			if which == 0 {
				avgsManager.RegActualChannels.RLock()
				for _, name := range spaces {
					select {
					case data := <-avgsManager.RegActualChannels.ChannelOut[name]:
						//fmt.Printf("%v gets %+v\n", name, data)
						//fmt.Printf("%+v\n", data)
						newData := JsonData{
							Space:   name,
							Type:    "current",
							Actuals: &data,
						}
						results = append(results, newData)
					default:
						// we do not wait for the channel to be ready
						//fmt.Println("data not ready, get skipped")
					}
				}
				avgsManager.RegActualChannels.RUnlock()

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

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(answer)
			return
		}

		// INFO: instead of using URL.string this could be done also with URL.Query, performance is the same but it breaks compatibility
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
			answer.Error = globals.Error.Error()
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(answer)

	})
}

func devicedefinitions() http.Handler {

	fn := func(a string, list []string) bool {
		for _, b := range list {
			if b == a {
				return true
			}
		}
		return false
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ApiManagerLog,
					mlogger.LoggerData{"apiManager.devicedefinitions",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()
		var rt JsonDefinitions

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			rt.Error = globals.InvalidOperation.Error()
			_ = json.NewEncoder(w).Encode(connectedSensors)
			return
		}

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		query := r.URL.Query()
		if cmd, ok := query["cmd"]; ok && len(cmd) == 1 {
			switch strings.ToLower(cmd[0]) {
			case "save":
				configFile, err := os.Open("./configuration.ini")
				if err != nil {
					rt.Error = globals.Error.Error()
					break
				}
				newConfigFile, err := os.Create("./temp.ini")
				if err != nil {
					rt.Error = globals.Error.Error()
					break
				}
				newSection := false
				sensorSection := false
				updated := false
				var writeError error

				scanner := bufio.NewScanner(configFile)
				for scanner.Scan() {
					newLine := scanner.Text()
					newSection = strings.Contains(newLine, "[")
					if newSection {
						sensorSection = strings.Contains(newLine, "[sensors]")
					}
					if !sensorSection && !strings.Contains(newLine, ";") {
						_, writeError = newConfigFile.WriteString(newLine + "\n")
					} else if !updated {
						_, writeError = newConfigFile.WriteString("[sensors]\n")
						updated = true
						if definitions, e := diskCache.ReadAllDefinitions(); e != nil {
							rt.Error = e.Error()
							break
						} else {
							for _, device := range definitions {
								var def string
								def = strings.Replace(device.Mac, ":", "", -1) + " : "
								def += strconv.Itoa(device.Id) + " "
								def += strconv.Itoa(int(device.MaxRate/1000000)) + " "
								if device.Bypass {
									def += "bypass "
								}
								if device.Report {
									def += "report "
								}
								if device.Enforce {
									def += "enforce "
								}
								if device.Strict {
									def += "strict "
								}
								def = strings.Trim(def, " ") + "\n"
								_, writeError = newConfigFile.WriteString(def)
							}
							_, writeError = newConfigFile.WriteString("\n")

						}
					}
					if writeError != nil {
						break
					}
				}
				if writeError != nil {
					_ = configFile.Close()
					_ = newConfigFile.Close()
					_ = os.Remove("temp.ini")
					rt.Error = globals.Error.Error()
				} else if err := scanner.Err(); err != nil {
					_ = configFile.Close()
					_ = newConfigFile.Close()
					_ = os.Remove("temp.ini")
					rt.Error = globals.Error.Error()
				} else {
					_ = configFile.Close()
					_ = newConfigFile.Close()
					_ = os.Rename("configuration.ini", "configuration_"+strconv.Itoa(int(time.Now().Unix()))+".ini")
					_ = os.Rename("temp.ini", "configuration.ini")
				}
			case "readall":
				var e error
				if rt.Definitions, e = diskCache.ReadAllDefinitions(); e != nil {
					rt.Error = e.Error()
				}
			case "read":
				if macQuery, ok := query["mac"]; ok {
					mac := macQuery[0]
					mac = strings.Replace(mac, ":", "", -1)
					if def, err := diskCache.ReadDefinition([]byte(mac)); err != nil {
						rt.Error = err.Error()
					} else {
						rt.Definitions = []dataformats.SensorDefinition{def}
					}
				}
			case "delete":
				if macQuery, ok := query["mac"]; ok {
					mac := macQuery[0]
					mac = strings.Replace(mac, ":", "", -1)
					if err := diskCache.DeleteDefinition([]byte(mac)); err != nil {
						rt.Error = err.Error()
					}
				} else {
					rt.Error = globals.InvalidOperation.Error()
				}
			case "add":
				rt.Error = globals.InvalidOperation.Error()
				if macQuery, ok := query["mac"]; ok {
					mac := macQuery[0]
					mac = strings.Replace(mac, ":", "", -1)
					if ids, ok := query["id"]; ok {
						if id, err := strconv.Atoi(ids[0]); err == nil {
							params := query["params"]
							definition := dataformats.SensorDefinition{
								Id:      id,
								Bypass:  fn("bypass", params),
								Report:  fn("report", params),
								Enforce: fn("enforce", params),
								Strict:  fn("strict", params),
							}
							// bypass has priority on strict
							definition.Strict = definition.Strict && !definition.Bypass
							// enforce does nothing if strict is given
							definition.Enforce = definition.Enforce && !definition.Strict
							if err := diskCache.WriteDefinition([]byte(mac), definition); err != nil {
								rt.Error = globals.Error.Error()
							} else {
								rt.Error = ""
							}
						}
					}
				}
			default:
				rt.Error = globals.InvalidOperation.Error()
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rt)

	})
}

func disconnectDevice() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ApiManagerLog,
					mlogger.LoggerData{"apiManager.disconnectDevice",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()
		var rt JsonCmdRt

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			rt.Error = globals.InvalidOperation.Error()
			_ = json.NewEncoder(w).Encode(connectedSensors)
			return
		}

		//Allow CORS here By * or specific origin
		if globals.DisableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		query := r.URL.Query()

		if macQuery, ok := query["mac"]; ok {
			mac := macQuery[0]
			mac = strings.Replace(mac, ":", "", -1)
			sensorManager.ActiveSensors.Lock()
			if chs, ok := sensorManager.ActiveSensors.Mac[mac]; ok {
				if chs.Tcp.Close() != nil {
					rt.Error = globals.Error.Error()
				}
			}
			sensorManager.ActiveSensors.Unlock()
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rt)

	})
}
