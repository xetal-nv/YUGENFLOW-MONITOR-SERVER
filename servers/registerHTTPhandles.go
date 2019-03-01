package servers

import (
	"countingserver/spaces"
	"countingserver/storage"
	"countingserver/support"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Register struct {
	Valid bool                 `json:"valid"`
	Error string               `json:"errorcode"`
	Data  *storage.SerieSample `json:"counter"`
}

type RegisterBank struct {
	Name string     `json:"space"`
	Data []Register `json:"counters"`
}

// handles single register requests
func singleRegisterHTTPhandles(path string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	tag := ""
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
		tag += strings.Replace(sp[i], "_", "", -1) + "_"
	}
	tag = tag[:len(tag)-1]
	rt := Register{true, "", new(storage.SerieSample)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("servers.singleRegisterHTTPhandles: recovering from: ", e)
				}
			}
		}()
		//fmt.Printf("%s %s %s \n", r.Method, r.URL, r.Proto)

		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		select {
		case data := <-spaces.LatestBankOut[sp[0]][sp[1]][sp[2]]:
			if e := rt.Data.Extract(data); e != nil {
				rt.Valid = false
				rt.Error = "ID"
			} else {
				rt.Data.Stag = tag
			}
		case <-time.After(2000 * time.Millisecond):
			rt.Valid = false
			rt.Error = "TO"
		}

		//noinspection GoUnhandledErrorResult
		json.NewEncoder(w).Encode(rt)
	})
}

// handles space requests
func spaceRegisterHTTPhandles(path string, als []string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	tag := ""
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
		tag += strings.Replace(sp[i], "_", "", -1) + "_"
	}
	tag = tag[:len(tag)-1]
	fmt.Println(tag)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("servers.singleRegisterHTTPhandles: recovering from: ", e)
				}
			}
		}()

		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		var rt RegisterBank

		rt.Name = tag

		for _, nm := range als {
			var tmp Register
			tmp.Data = new(storage.SerieSample)
			select {
			case data := <-spaces.LatestBankOut[sp[0]][sp[1]][nm]:
				if e := tmp.Data.Extract(data); e != nil {
					tmp.Valid = false
					tmp.Error = "ID"
				} else {
					tmp.Valid = true
				}
				tmp.Data.Stag = tag + "_" + strings.Replace(nm, "_", "", -1)
			case <-time.After(2000 * time.Millisecond):
				tmp.Valid = false
				tmp.Error = "TO"
			}
			rt.Data = append(rt.Data, tmp)
		}

		//noinspection GoUnhandledErrorResult
		json.NewEncoder(w).Encode(rt)

	})
}

// TODO handles datatype requests
func datatypeRegisterHTTPhandles(path string, als [][]string) http.Handler {
	return nil
}
