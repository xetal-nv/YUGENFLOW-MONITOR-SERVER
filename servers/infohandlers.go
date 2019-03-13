package servers

import (
	"countingserver/gates"
	"countingserver/spaces"
	"countingserver/support"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// returns the DevLog
func dvlHTTHandler() http.Handler {
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
	fmt.Println("go")
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

		_ = json.NewEncoder(w).Encode(inst)
	})
}
