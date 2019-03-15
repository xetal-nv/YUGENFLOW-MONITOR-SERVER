package servers

import (
	"countingserver/storage"
	"countingserver/support"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var cmds = []string{"last", "type", "space", "analysis", "start", "end"}

// http://localhost:8090/series?type=sample?space=noname?analysis=current?start=12345?end=12345

// returns the DevLog
func seriesHTTPhandler() http.Handler {

	params := make(map[string]string)
	for _, i := range cmds {
		params[i] = ""
	}

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

		for _, rp := range strings.Split(r.URL.String(), "?")[1:] {
			val := strings.Split(rp, "=")
			if _, ok := params[strings.Trim(val[0], " ")]; ok {
				params[strings.Trim(val[0], " ")] = strings.Trim(val[1], " ")
			} else {
				go func() {
					support.DLog <- support.DevData{"servers.seriesHTTPhandler: " + strings.Trim(val[0], " "),
						support.Timestamp(), "illegal request", []int{1}, true}
				}()
				return
			}
		}

		label := support.StringLimit(params["type"], support.LabelLength)
		label += support.StringLimit(params["space"], support.LabelLength)
		label += support.StringLimit(params["analysis"], support.LabelLength)
		if params["last"] != "" {
			if num, e := strconv.Atoi(params["last"]); e == nil {
				var s storage.SampleData
				switch params["type"] {
				case "sample":
					s = &storage.SerieSample{Stag: label, Sts: support.Timestamp()}
				case "entry":
					s = &storage.SerieEntries{Stag: label, Sts: support.Timestamp()}
				default:
					return
				}
				if tag, ts, vals, e := storage.ReadLastN(s, num, params["analysis"] != "current"); e == nil {
					// TODO format into JSON !!
					for _, v := range s.UnmarshalSliceSS(tag, ts, vals) {
						fmt.Fprintf(w, "%v\n", v)
					}
				}
			}
		} else {
			// TODO HERE range series
			fmt.Fprintf(w, "range read %v\n", label)
		}
	})
}
