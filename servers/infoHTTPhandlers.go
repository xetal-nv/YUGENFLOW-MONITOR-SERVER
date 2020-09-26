package servers

import (
	"encoding/json"
	"fmt"
	"gateserver/gates"
	"gateserver/spaces"
	"gateserver/supp"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// returns the DevLog
func dvlHTTHandler() http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"dvlHTTHandler",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("dvlHTTHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		//noinspection GoUnhandledErrorResult
		supp.DLog <- supp.DevData{Tag: "read"}
		_, _ = fmt.Fprintf(w, <-supp.ODLog)
	})
}

type JsonDevs struct {
	Id       int  `json:"deviceid"`
	Reversed bool `json:"reversed"`
}

type JsonGate struct {
	Id      int        `json:"gateid"`
	Devices []JsonDevs `json:"devices"`
}

type JsonEntry struct {
	Id    int        `json:"entryid"`
	Gates []JsonGate `json:"gates"`
}

type JsonSpace struct {
	Id      string      `json:"spacename"`
	Entries []JsonEntry `json:"entries"`
}

var inst []JsonSpace

// returns the installation information
func infoHTTHandler() http.Handler {
	for spn, spd := range spaces.SpaceDef {
		var space JsonSpace
		space.Id = strings.Replace(spn, "_", "", -1)
		for _, enm := range spd {
			var entry JsonEntry
			entry.Id = enm
			for gnm, gnd := range gates.EntryList[enm].Gates {
				var gate JsonGate
				gate.Id = gnm
				for _, dvn := range gnd {
					var device JsonDevs
					device.Id = dvn
					device.Reversed = gates.EntryList[enm].SenDef[dvn].Reversed
					gate.Devices = append(gate.Devices, device)
				}
				entry.Gates = append(entry.Gates, gate)
			}
			space.Entries = append(space.Entries, entry)
		}
		inst = append(inst, space)
	}

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"infoHTTHandler",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("infoHTTHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		_ = json.NewEncoder(w).Encode(inst)

	})
}

type JsonRt struct {
	Name      string `json:"name"`
	Qualifier string `json:"qualifier"`
}

// returns the analysis definition information
func asysHTTHandler() http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	var asys []JsonRt
	if dt := os.Getenv("ANALYSISPERIOD"); dt != "" {
		for _, v := range strings.Split(strings.Trim(dt, ";"), ";") {
			vc := strings.Split(strings.Trim(v, " "), " ")
			//if len(vc) == 2 || len(vc) == 4 {
			if len(vc) == 2 {
				el := JsonRt{strings.Trim(vc[0], " "), strings.Trim(vc[1], " ")}
				asys = append(asys, el)
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"dvlHTTHandler",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("dvlHTTHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		_ = json.NewEncoder(w).Encode(asys)

	})
}

// returns the installation space geometry as SVG
func planHTTPHandler(name string) http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	empty, err := ioutil.ReadFile("./installation/empty.svg")
	if err != nil {
		log.Fatal("planHTTPHandler: Fatal error, resource empty.svg is missing")
	}

	rt := JsonRt{Name: name}
	if name != "" {
		data, err := ioutil.ReadFile("./installation/" + name + ".svg")
		if err != nil {
			log.Printf("planHTTPHandler %v: error reading svg file, empty one assigned\n", name)
			rt.Qualifier = string(empty)
		} else {
			rt.Qualifier = string(data)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"planHTTPHandler " + name,
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("planHTTPHandler "+name+": recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		_ = json.NewEncoder(w).Encode(rt)

	})
}

type JsonUnd struct {
	Id  int    `json:"id"`
	Mac string `json:"mac"`
}

// returns the list of connected unused devices
func unusedDeviceHTTPHandler() http.Handler {

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"HandlerFunc",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("HandlerFunc: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		var und []JsonUnd
		mutexUnusedDevices.RLock()
		for id, mac := range unusedDevice {
			mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", []byte(mac)), " ", ":", -1), ":")
			und = append(und, JsonUnd{id, mach})
		}
		mutexUnusedDevices.RUnlock()

		_ = json.NewEncoder(w).Encode(und)

	})
}

type JsonUdef struct {
	Mac       string `json:"mac"`
	State     bool   `json:"redefined"`
	Connected bool   `json:"connected"`
}

// returns the list of connected undefined devices
func undefinedDeviceHTTPHandler(opt string) http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"undefinedDeviceHTTPHandler",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("undefinedDeviceHTTPHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		var und []JsonUdef
		mutexUnknownMac.RLock()
		for mac, st := range unknownDevice {
			mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", []byte(mac)), " ", ":", -1), ":")
			_, cn := unknownMacChan[mac]
			switch opt {
			case "undefined":
				if !st {
					und = append(und, JsonUdef{mach, st, cn})
				}
			case "defined":
				if st {
					und = append(und, JsonUdef{mach, st, cn})
				}
			case "active":
				if cn {
					und = append(und, JsonUdef{mach, st, cn})
				}
			case "notactive":
				if !cn {
					und = append(und, JsonUdef{mach, st, cn})
				}
			default:
				und = append(und, JsonUdef{mach, st, cn})
			}
		}
		mutexUnknownMac.RUnlock()

		_ = json.NewEncoder(w).Encode(und)
	})
}

// returns the list of connected used devices
func usedDeviceHTTPHandler() http.Handler {

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"HandlerFunc",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("HandlerFunc: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		var udev []JsonUnd
		mutexSensorMacs.RLock()
		for id, mac := range sensorMacID {
			mach := strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", ":", -1), ":")
			udev = append(udev, JsonUnd{id, mach})
		}
		mutexSensorMacs.RUnlock()

		_ = json.NewEncoder(w).Encode(udev)

	})
}

// returns list of connected devices pending approval

func pendingDeviceHTTPHandler() http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"pendingDeviceHTTPHandler",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("pendingDeviceHTTPHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		mutexPendingDevices.Lock()
		for mac := range pendingDevice {
			if pendingDevice[mac] {
				_, _ = fmt.Fprintf(w, strings.Trim(strings.Replace(fmt.Sprintf("% x ", mac), " ", ":", -1), ":"))
				_, _ = fmt.Fprintf(w, "\n")
			}
		}
		mutexPendingDevices.Unlock()

	})
}

// it is the kill switch

func killswitchHTTPHandler() http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"pendingDeviceHTTPHandler",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("pendingDeviceHTTPHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		_, _ = fmt.Fprintf(w, "Server stopped, wait for restart\n")

		go func() {
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()

	})
}
