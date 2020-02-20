package servers

import (
	"encoding/json"
	"gateserver/support"
	"log"
	"net/http"
	"os"
	"strings"
)

type JsonCmdRt struct {
	Rt    string `json:"answer"`
	State bool   `json:"State"`
}

// handles the commands to the sensors
func commandHTTHandler() http.Handler {

	//params := make(map[string]string)

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"servers.commandHTTHandler",
						support.Timestamp(), "recovering", []int{1}, true}
				}()
				log.Println("servers.commandHTTHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		params := make(map[string]string)
		for _, i := range commandNames {
			params[i] = ""
		}
		// command pin is not exposed to the user with the list command
		params["pin"] = ""

		for _, rp := range strings.Split(r.URL.String(), "?")[1:] {
			val := strings.Split(rp, "=")
			if _, ok := params[strings.Trim(val[0], " ")]; ok {
				params[strings.Trim(val[0], " ")] = strings.Trim(val[1], " ")
			} else {
				go func() {
					support.DLog <- support.DevData{"servers.commandHTTHandler: " + strings.Trim(val[0], " "),
						support.Timestamp(), "illegal request", []int{1}, true}
				}()
				return
			}
		}

		if params["pin"] != "" {
			if params["pin"] == pinDbg {
				ip := strings.Split(strings.Replace(r.RemoteAddr, "[::1]", "localhost", 1), ":")[0]
				dbgMutex.Lock()
				dbgRegistry[ip] = support.Timestamp()
				dbgMutex.Unlock()
				_ = json.NewEncoder(w).Encode(JsonCmdRt{"", true})
			} else {
				_ = json.NewEncoder(w).Encode(JsonCmdRt{"", false})
			}
			return
		}

		if params["id"] != "" && params["mac"] != "" {
			_ = json.NewEncoder(w).Encode(JsonCmdRt{"error: illegal command providing both mac and id", false})
			return
		}

		if params["async"] != "1" {
			rv := exeParamCommand(params)
			_ = json.NewEncoder(w).Encode(rv)
		} else {
			go func() { exeParamCommand(params) }()
			_ = json.NewEncoder(w).Encode(JsonCmdRt{"async mode", true})
		}
	})
}
