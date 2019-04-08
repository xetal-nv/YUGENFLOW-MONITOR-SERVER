package servers

import (
	"encoding/json"
	"gateserver/support"
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

// index for the map[string]string argument of executeCommand
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
					support.DLog <- support.DevData{"commandHTTHandler",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("commandHTTHandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		params := make(map[string]string)
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

		rv := executeCommand(params)

		if params["async"] != "1" {
			_ = json.NewEncoder(w).Encode(rv)
		}
	})
}

// execute a command towards a sensor as specified by the params map
// see cmds definition for what parameters are allowed
func executeCommand(params map[string]string) (rv Jsoncmdrt) {
	rv = Jsoncmdrt{"", false}
	if params["cmd"] != "" || params["id"] != "" {
		if params["cmd"] == "list" {
			//fmt.Println("CMD: LIST")
			keys := ""
			for k := range cmdAPI {
				keys += k + ", "
			}
			rv.Rt = keys[:len(keys)-2]
			rv.State = true
			params["async"] = "0"
		} else if id, e := strconv.Atoi(params["id"]); e == nil {
			//fmt.Println("CMD: NOT LIST: ", params)
			if _, ok := SensorCmd[id]; ok {
				//fmt.Println("CMD: found CMD channel")
				if v, ok := cmdAPI[params["cmd"]]; ok {
					//fmt.Println("CMD: accepted CMD", cmdAPI[params["cmd"]])
					var to int
					if to, e = strconv.Atoi(params["timeout"]); e != nil || params["timeout"] == "" {
						to = timeout
					}
					cmd := []byte{v.cmd}
					// need to execute the command on sensor with the given ID
					if v.lgt != 0 && params["val"] != "" {
						par := strings.Split(params["val"][1:len(params["val"])-1], ",")
						//fmt.Println("CMD: found PARAMS",par)
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
							rv.Rt = "insufficient parameters"
						}
					}
					if cmd != nil {
						//fmt.Println("CMD: Executing command")
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
	return
}

// execute command CMD with parameter val on sensor ID. all values are strings
func exeCommand(id, cmd string, val []int) Jsoncmdrt {
	params := make(map[string]string)
	for _, i := range cmds {
		params[i] = ""
	}
	if v, e := json.Marshal(val); e != nil {
		return Jsoncmdrt{"", false}
	} else {
		params["val"] = string(v)
	}
	params["cmd"] = cmd
	params["id"] = id
	if support.Debug > 0 {
		log.Printf("exeCommand received and executing %v\n", params)
	}
	return executeCommand(params)
}
