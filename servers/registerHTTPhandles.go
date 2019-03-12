package servers

import (
	"countingserver/spaces"
	"countingserver/support"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type Register struct {
	Valid bool        `json:"valid"`
	Error string      `json:"errorcode"`
	Data  GenericData `json:"counter"`
}

type RegisterBank struct {
	Name string     `json:"id"`
	Data []Register `json:"counters"`
}

// handles single register requests
func singleRegisterHTTPhandler(path string, ref string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	tag := ""
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
		tag += strings.Replace(sp[i], "_", "", -1) + "_"
	}
	tag = tag[:len(tag)-1]
	rt := Register{true, "", dataMap[ref]()}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"servers.singleRegisterHTTPhandler: recovering server",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("servers.singleRegisterHTTPhandler: died from: ", e)
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
				rt.Data.SetTag(tag)
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
func spaceRegisterHTTPhandler(path string, als []string, ref string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	tag := ""
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
		tag += strings.Replace(sp[i], "_", "", -1) + "_"
	}
	tag = tag[:len(tag)-1]

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"servers.spaceRegisterHTTPhandler: recovering server",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("servers.spaceRegisterHTTPhandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		_ = json.NewEncoder(w).Encode(retrieveSpace(tag, sp, als, ref))

	})
}

func datatypeRegisterHTTPhandler(path string, rg map[string][]string) http.Handler {
	tag := strings.Replace(path[1:], "_", "", -1)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"servers.datatypeRegisterHTTPhandler: recovering server",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("servers.datatypeRegisterHTTPhandler: died from: ", e)
			}
		}()
		//fmt.Printf("%s %s %s \n", r.Method, r.URL, r.Proto)

		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		var rt []RegisterBank
		for sp, als := range rg {
			sp0 := support.StringLimit(path[1:], support.LabelLength)
			rt = append(rt, retrieveSpace(tag, []string{sp0, sp}, als, tag))
		}

		_ = json.NewEncoder(w).Encode(rt)
	})
}

func retrieveSpace(tag string, sp []string, als []string, ref string) (rt RegisterBank) {
	//fmt.Println(tag, sp, als, ref)
	rt.Name = tag

	var ca []chan Register

	for _, nm := range als {
		c := make(chan Register)
		ca = append(ca, c)
		go func(c chan Register, nm string) {
			var tmp Register
			tmp.Data = dataMap[ref]()
			tmp.Data.SetTag(tag + "_" + strings.Replace(nm, "_", "", -1))
			select {
			case data := <-spaces.LatestBankOut[sp[0]][sp[1]][nm]:
				if e := tmp.Data.Extract(data); e != nil {
					tmp.Valid = false
					tmp.Error = "ID"
				} else {
					tmp.Valid = true
				}
			case <-time.After(2000 * time.Millisecond):
				tmp.Valid = false
				tmp.Error = "TO"
			}
			c <- tmp
		}(c, nm)
	}

	for _, cai := range ca {
		v := <-cai
		rt.Data = append(rt.Data, v)
	}
	return
}
