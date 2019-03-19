package servers

import (
	"countingserver/gates"
	"countingserver/spaces"
	"countingserver/support"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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
					support.DLog <- support.DevData{"dvlHTTHandler",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("dvlHTTHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		//noinspection GoUnhandledErrorResult
		support.DLog <- support.DevData{Tag: "read"}
		_, _ = fmt.Fprintf(w, <-support.ODLog)
	})
}

type Jsondevs struct {
	Id       int  `json:"deviceid"`
	Reversed bool `json:"reversed"`
}

type Jsongate struct {
	Id      int        `json:"gateid"`
	Devices []Jsondevs `json:"devices"`
}

type Jsonentry struct {
	Id    int        `json:"entryid"`
	Gates []Jsongate `json:"gates"`
}

type Jsonspace struct {
	Id      string      `json:"spacename"`
	Entries []Jsonentry `json:"entries"`
}

var inst []Jsonspace

// returns the installation information
func infoHTTHandler() http.Handler {
	for spn, spd := range spaces.SpaceDef {
		var space Jsonspace
		space.Id = strings.Replace(spn, "_", "", -1)
		for _, enm := range spd {
			var entry Jsonentry
			entry.Id = enm
			for gnm, gnd := range gates.EntryList[enm].Gates {
				var gate Jsongate
				gate.Id = gnm
				for _, dvn := range gnd {
					var device Jsondevs
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
					support.DLog <- support.DevData{"infoHTTHandler",
						support.Timestamp(), "", []int{1}, true}
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

type Jsonrt struct {
	Name      string `json:"name"`
	Qualifier string `json:"qualifier"`
}

func asysHTTHandler() http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	var asys []Jsonrt
	if dt := os.Getenv("SAVEWINDOW"); dt != "" {
		for _, v := range strings.Split(strings.Trim(dt, ";"), ";") {
			vc := strings.Split(strings.Trim(v, " "), " ")
			if len(vc) == 2 {
				el := Jsonrt{strings.Trim(vc[0], " "), strings.Trim(vc[1], " ")}
				asys = append(asys, el)
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"dvlHTTHandler",
						support.Timestamp(), "", []int{1}, true}
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

func planHTTPHandler(name string) http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	rt := Jsonrt{Name: name}
	if name != "" {
		data, err := ioutil.ReadFile("./installation/" + name + ".svg")
		if err != nil {
			log.Printf("planHTTPHandler %v: error %v reading svg file\n", name, err)
		}
		rt.Qualifier = string(data)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"planHTTPHandler " + name,
						support.Timestamp(), "", []int{1}, true}
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
