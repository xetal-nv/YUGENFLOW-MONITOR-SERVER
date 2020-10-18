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
				mlogger.Recovered(globals.ClientManagerLog,
					mlogger.LoggerData{"clientManager.info",
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

		_ = json.NewEncoder(w).Encode(installationInfo)
		gateManager.GateStructure.RUnlock()
		entryManager.EntryStructure.RUnlock()
		spaceManager.SpaceStructure.RUnlock()

	})
}

func connectedSensors() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				mlogger.Recovered(globals.ClientManagerLog,
					mlogger.LoggerData{"clientManager.connectedSensors",
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
				mlogger.Recovered(globals.ClientManagerLog,
					mlogger.LoggerData{"clientManager.invalidSensors",
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
				mlogger.Recovered(globals.ClientManagerLog,
					mlogger.LoggerData{"clientManager.measurementDefinitions",
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

func latestData(all, useDB bool, which int) http.Handler {
	// which :
	// 0 - all
	// 1 - real time / real
	// 2- reference

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				// TODOD remove
				fmt.Println(e)
				mlogger.Recovered(globals.ClientManagerLog,
					mlogger.LoggerData{"clientManager.invalidSensors",
						"route terminated and recovered unexpectedly",
						[]int{1}, true})
			}
		}()

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
		if !useDB {
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

		} else {
			numberSamples := 1
			for _, rp := range strings.Split(r.URL.String(), "?")[1:] {
				if val, err := strconv.Atoi(rp); err == nil {
					numberSamples = val
				}
			}
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
							Type:    "real",
							Results: make(map[string][]dataformats.MeasurementSample),
						}
						for _, val := range data {
							newData.Results["flows"] = append(newData.Results[val.Qualifier], val)
						}
						results = append(results, newData)
						fmt.Println(data)
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(results)

	})
}

//func register() http.Handler {
//
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		defer func() {
//			if e := recover(); e != nil {
//				//fmt.Println(e)
//				mlogger.Recovered(globals.ClientManagerLog,
//					mlogger.LoggerData{"clientManager.register",
//						"route terminated and recovered unexpectedly",
//						[]int{1}, true})
//			}
//		}()
//
//		//Allow CORS here By * or specific origin
//		if globals.DisableCORS {
//			w.Header().Set("Access-Control-Allow-Origin", "*")
//		}
//
//		vars := mux.Vars(r)
//
//		mac, err := verifyDevice(vars["id"])
//		for i := 2; i < len(mac); i += 3 {
//			mac = mac[:i] + ":" + mac[i:]
//		}
//		if err == nil {
//			_, err = getRegistrantEmail(mac)
//		}
//
//		w.WriteHeader(http.StatusOK)
//		rt := dataformats.ApiResponse{
//			Result: mac,
//		}
//		if err != nil {
//			rt.Error = err.Error()
//		}
//		_ = json.NewEncoder(w).Encode(rt)
//		return
//	})
//}
//

//func executeLink() http.Handler {
//
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		defer func() {
//			if e := recover(); e != nil {
//				mlogger.Recovered(globals.ClientManagerLog,
//					mlogger.LoggerData{"clientManager.executeLink",
//						"route terminated unexpectedly",
//						[]int{1}, true})
//			}
//		}()
//		//Allow CORS here By * or specific origin
//		if globals.DisableCORS {
//			w.Header().Set("Access-Control-Allow-Origin", "*")
//		}
//
//		vars := mux.Vars(r)
//		if _, err := verifyDevice(vars["id"]); err == nil {
//			if cm, err := cache.TemporaryLinks.Get(vars["cmd"]); err == nil {
//				//fmt.Println("ok", string(cm))
//				if res, err := http.Get(string(cm)); err == nil {
//					var rv dataformats.ApiResponse
//					data, _ := ioutil.ReadAll(res.Body)
//					_ = json.Unmarshal(data, &rv)
//					//fmt.Println(rv)
//					_ = cache.TemporaryLinks.Delete(vars["cmd"])
//					w.WriteHeader(http.StatusOK)
//					_ = json.NewEncoder(w).Encode(rv)
//					return
//				}
//			}
//		}
//		w.WriteHeader(http.StatusOK)
//		_ = json.NewEncoder(w).Encode(dataformats.ApiResponse{
//			Error: globals.ApiError.Error(),
//		})
//	})
//}
//
//func deviceCommandLink(cm string) http.Handler {
//
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		defer func() {
//			if e := recover(); e != nil {
//				mlogger.Recovered(globals.ClientManagerLog,
//					mlogger.LoggerData{"clientManager.deviceCommandLink",
//						"route terminated unexpectedly",
//						[]int{1}, true})
//				fmt.Println(e)
//
//			}
//		}()
//
//		//Allow CORS here By * or specific origin
//		if globals.DisableCORS {
//			w.Header().Set("Access-Control-Allow-Origin", "*")
//		}
//
//		vars := mux.Vars(r)
//		var rt dataformats.ApiResponse
//		if mac, err := verifyDevice(vars["id"]); err == nil && mac != "" {
//			// bypassed for development
//			if email, err := getRegistrantEmail(mac); err == nil || globals.DebugActive {
//				if globals.DebugActive {
//					email = globals.SupportEmail
//				}
//				command := cm
//				switch cm {
//				case "block":
//					command += "/" + mac
//				case "reset":
//					command += "/" + mac
//				case "resetIdentifier":
//					command += "/" + mac
//				case "mode":
//					command += "/" + mac + "/" + vars["mode"]
//				case "localadaptation":
//					command += "/" + mac + "/" + vars["mode"]
//				default:
//					mlogger.Warning(globals.ClientManagerLog,
//						mlogger.LoggerData{"clientManager.deviceCommandLink",
//							"illegal API command for device " + mac,
//							[]int{1}, true})
//				}
//				linkId, err := generateLink(command)
//				if err != nil {
//					rt.Error = err.Error()
//				} else {
//					err = sendLink(email, globals.CommandServer+"/yugenface/"+vars["id"]+"/"+linkId, mac, cm)
//					if err != nil {
//						rt.Error = globals.EmailFailed.Error()
//					}
//				}
//			} else {
//				rt.Error = err.Error()
//			}
//		} else {
//			rt.Error = globals.ApiError.Error()
//		}
//
//		w.WriteHeader(http.StatusOK)
//		_ = json.NewEncoder(w).Encode(rt)
//		return
//	})
//}
//
//func deviceCommand(cm string) http.Handler {
//
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		defer func() {
//			if e := recover(); e != nil {
//				mlogger.Recovered(globals.ClientManagerLog,
//					mlogger.LoggerData{"clientManager.deviceCommand",
//						"route terminated unexpectedly",
//						[]int{1}, true})
//				fmt.Println(e)
//
//			}
//		}()
//
//		//Allow CORS here By * or specific origin
//		if globals.DisableCORS {
//			w.Header().Set("Access-Control-Allow-Origin", "*")
//		}
//
//		vars := mux.Vars(r)
//		var rt dataformats.ApiResponse
//		if mac, err := verifyDevice(vars["id"]); err == nil {
//			if _, err := getRegistrantEmail(mac); err == nil || globals.DebugActive {
//				command := cm
//				switch cm {
//				case "configuration":
//					command += "/" + mac
//				case "result":
//					command += "/" + mac + "/" + vars["howmany"]
//				default:
//					mlogger.Warning(globals.ClientManagerLog,
//						mlogger.LoggerData{"clientManager.deviceCommand",
//							"illegal API command for device " + mac,
//							[]int{1}, true})
//				}
//				if res, err := http.Get(globals.APIServer + "/" + command); err == nil {
//					var rv dataformats.ApiResponse
//					data, _ := ioutil.ReadAll(res.Body)
//					_ = json.Unmarshal(data, &rv)
//					_ = cache.TemporaryLinks.Delete(vars["cmd"])
//					w.WriteHeader(http.StatusOK)
//					_ = json.NewEncoder(w).Encode(rv)
//					return
//				}
//			} else {
//				rt.Error = err.Error()
//			}
//		} else {
//			rt.Error = err.Error()
//		}
//
//		w.WriteHeader(http.StatusOK)
//		_ = json.NewEncoder(w).Encode(rt)
//		return
//	})
//}
