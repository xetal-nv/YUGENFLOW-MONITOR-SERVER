package servers

import (
	"encoding/json"
	"fmt"
	"gateserver/spaces"
	"gateserver/support"
	"log"
	"net/http"
	"os"
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

const ito = 4000

// handles single register requests
func singleRegisterHTTPhandler(path string, ref string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	tag := ""
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
		tag += strings.Replace(sp[i], "_", "", -1) + "_"
	}
	tag = tag[:len(tag)-1]
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"servers.singleRegisterHTTPhandler: recovering server",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("servers.singleRegisterHTTPhandler: died from: ", e)
				fmt.Println("servers.singleRegisterHTTPhandler: died from: ", e)
			}
		}()
		//fmt.Printf("%s %s %s \n", r.Method, r.URL, r.Proto)

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// check if the client is authorised to access debug data
		//ip := strings.Split(strings.Replace(r.RemoteAddr, "[::1]", "localhost", 1), ":")[0]
		//fmt.Println(r.RemoteAddr, ip)
		//authorised := false
		//dbgMutex.RLock()
		//if tts, ok := dbgRegistry[ip]; ok {
		//	if (support.Timestamp()-tts)/1000 <= (authDbgInterval * 60) {
		//		authorised = true
		//	} else {
		//		delete(dbgRegistry, ip)
		//	}
		//}
		//dbgMutex.RUnlock()
		//if authorised {
		//	log.Println("servers.seriesHTTPhandler: answering (debug) authorised device at address:", ip)
		//}

		rt := Register{true, "", dataMap[ref]()}

		// fmt.Println(sp, ref)

		//if authorised || (!authorised && sp[0] != "entry___") {
		if spaces.LatestBankOut[sp[0]][sp[1]][sp[2]] == nil {
			log.Println("HTTP got a nil channel")
			// fmt.Println("HTTP got a nil channel")
		}
		select {
		case data := <-spaces.LatestBankOut[sp[0]][sp[1]][sp[2]]:
			// fmt.Println(data)
			if data != nil {
				if e := rt.Data.Extract(data); e != nil {
					// fmt.Println("1")
					rt.Valid = false
					rt.Error = "ID"
				} else {
					// fmt.Println(rt.Data)
					rt.Data.SetTag(tag)
				}
			} else {
				// fmt.Println("3")
				rt.Valid = false
				rt.Error = "NIL"
			}
		case <-time.After(ito * time.Millisecond):
			if spaces.LatestBankOut[sp[0]][sp[1]][sp[2]] == nil {
				log.Println("HTTP got a nil channel")
			}
			rt.Valid = false
			rt.Error = "TO"
		}
		//} else {
		//	rt.Valid = false
		//}
		// fmt.Println(rt.Data)

		//noinspection GoUnhandledErrorResult
		json.NewEncoder(w).Encode(rt)
		//fmt.Println("http sent", rt, r.URL)

	})
}

// handles requests for all current data for a given space
func spaceRegisterHTTPhandler(path string, als []string, ref string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	tag := ""
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
		tag += strings.Replace(sp[i], "_", "", -1) + "_"
	}
	tag = tag[:len(tag)-1]

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

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
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// check if the client is authorised to access debug data
		//ip := strings.Split(strings.Replace(r.RemoteAddr, "[::1]", "localhost", 1), ":")[0]
		//authorised := false
		//dbgMutex.RLock()
		//if tts, ok := dbgRegistry[ip]; ok {
		//	if (support.Timestamp()-tts)/1000 <= (authDbgInterval * 60) {
		//		authorised = true
		//	} else {
		//		delete(dbgRegistry, ip)
		//	}
		//}
		//dbgMutex.RUnlock()
		//if authorised {
		//	log.Println("servers.spaceRegisterHTTPhandler: answering (debug) authorised device at address:", ip)
		//}

		//if authorised || (!authorised && sp[0] != "entry___") {

		_ = json.NewEncoder(w).Encode(retrieveSpace(tag, sp, als, ref))
		//} else {
		//	_ = json.NewEncoder(w).Encode(Register{false, "", dataMap[ref]()})
		//}

	})
}

// handles requests for all current data for a given type (sample, entry)
func datatypeRegisterHTTPhandler(path string, rg map[string][]string) http.Handler {
	tag := strings.Replace(path[1:], "_", "", -1)

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

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

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// check if the client is authorised to access debug data
		//ip := strings.Split(strings.Replace(r.RemoteAddr, "[::1]", "localhost", 1), ":")[0]
		//authorised := false
		//dbgMutex.RLock()
		//if tts, ok := dbgRegistry[ip]; ok {
		//	if (support.Timestamp()-tts)/1000 <= (authDbgInterval * 60) {
		//		authorised = true
		//	} else {
		//		delete(dbgRegistry, ip)
		//	}
		//}
		//dbgMutex.RUnlock()
		//if authorised {
		//	log.Println("servers.datatypeRegisterHTTPhandler: answering (debug) authorised device at address:", ip)
		//}

		var rt []RegisterBank
		//if authorised || (!authorised && path != "/entry") {

		for sp, als := range rg {
			sp0 := support.StringLimit(path[1:], support.LabelLength)
			rt = append(rt, retrieveSpace(tag, []string{sp0, sp}, als, tag))
		}
		//}

		_ = json.NewEncoder(w).Encode(rt)
	})
}

// support function for topology extraction
func retrieveSpace(tag string, sp []string, als []string, ref string) (rt RegisterBank) {
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
				if data != nil {
					if e := tmp.Data.Extract(data); e != nil {
						tmp.Valid = false
						tmp.Error = "ID"
					} else {
						tmp.Valid = true
					}
				} else {
					tmp.Valid = false
					tmp.Error = "NIL"
				}
			case <-time.After(ito * time.Millisecond):
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
