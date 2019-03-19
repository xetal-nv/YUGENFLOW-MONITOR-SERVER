package servers

import (
	"countingserver/support"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Jsoncmdrt struct {
	Rt    string `json:"answer"`
	State bool   `json:"State"`
}

type cmdspecs struct {
	cmd byte
	lgt int
}

// handles the commands to the sensors
func commandHTTHandler() http.Handler {
	var cmds = []string{"cmd", "val", "async", "id", "timeout"}
	var cmdAPI = map[string]cmdspecs{ // provides length for legal server2gate commands
		"srate":     {2, 1},
		"savg":      {3, 1},
		"bgth":      {4, 2},
		"occth":     {5, 2},
		"rstbg":     {6, 0},
		"readdiff":  {7, 0},
		"resetdiff": {8, 0},
		"readinc":   {9, 0},
		"rstinc":    {10, 0},
		"readoutc":  {11, 0},
		"rstoutc":   {12, 0},
		"readid":    {13, 0},
		"setid":     {14, 2},
	}

	params := make(map[string]string)

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"commandHTTHandler",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("commandHTTHandler: recovering from: ", e)
			}
		}()

		rv := Jsoncmdrt{"", false}

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		for _, i := range cmds {
			params[i] = ""
		}
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
		if params["cmd"] != "" || params["id"] != "" {
			if params["cmd"] == "list" {
				keys := ""
				for k := range cmdAPI {
					keys += k + ", "
				}
				rv.Rt = keys[:len(keys)-2]
				rv.State = true
				params["async"] = "0"
			} else if id, e := strconv.Atoi(params["id"]); e == nil {
				if _, ok := SensorCmd[id]; ok {
					if v, ok := cmdAPI[params["cmd"]]; ok {
						var to int
						if to, e = strconv.Atoi(params["timeout"]); e != nil || params["timeout"] == "" {
							to = timeout
						}
						cmd := []byte{v.cmd}
						// need to execute the command on sensor with the given ID
						if v.lgt != 0 && params["val"] != "" {
							par := strings.Split(params["val"][1:len(params["val"])-1], ",")
							if v.lgt == len(par) {
								for _, val := range par {
									if data, err := strconv.Atoi(strings.Trim(val, " ")); err != nil || data > 255 {
										cmd = nil
										break
									} else {
										cmd = append(cmd, byte(data))
									}
								}
							} else {
								cmd = nil
								rv.Rt = "error"
							}
						} else if params["val"] == "" {
							cmd = nil
							rv.Rt = "error"
						}
						if cmd != nil {
							select {
							case SensorCmd[id] <- cmd:
								rv.State = true
								select {
								case rt := <-SensorCmd[id]:
									rv.Rt = string(rt)
								case <-time.After(time.Duration(to) * time.Second):
									rv.Rt = "to"
									// timeout to be used on the sending side to remove a possible hanging goroutine
								}
							case <-time.After(time.Duration(to) * time.Second):
								rv.Rt = "to"
							}
						}
					}
				}
			}
		}
		if params["async"] != "1" {
			_ = json.NewEncoder(w).Encode(rv)
		}
	})
}
