package servers

import (
	"encoding/json"
	"fmt"
	"gateserver/storage"
	"gateserver/support"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// returns a series of data. it accepts the following parameters
// type : data type
// space : space name
// analysis : analysis name
// start : initial time stamp of the range
// end : last time stamp of the range
// last : number of last samples to return
// last and (start.end) are mutually exclusive with last having priority in case they are all specified
func seriesHTTPhandler() http.Handler {
	var cmds = []string{"last", "type", "space", "analysis", "start", "end"}

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	//dbgRegistry["192.168.1.116"] = support.Timestamp()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"servers,seriesHTTPhandler",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("servers.seriesHTTPhandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// check if the client is authorised to access debug data
		//ip := strings.Split(r.RemoteAddr,":")[0]
		//authorised := false
		//dbgMutex.Lock()
		//if _, ok := dbgRegistry[ip]; ok {
		//fmt.Println((support.Timestamp() - tts)/1000<(authInterval * 60))
		//authorised = true
		//}
		//if authorised {
		//	fmt.Println("servers.seriesHTTPhandler: answering authorised device at address ", ip)
		//}
		//fmt.Println(authorised, ip)

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
					support.DLog <- support.DevData{"servers.seriesHTTPhandler: " + strings.Trim(val[0], " "),
						support.Timestamp(), "illegal request", []int{1}, true}
				}()
				_, _ = fmt.Fprintf(w, "")
				return
			}
		}

		label := support.StringLimit(params["type"], support.LabelLength)
		label += support.StringLimit(params["space"], support.LabelLength)
		label += support.StringLimit(params["analysis"], support.LabelLength)
		if params["last"] != "" {
			var s storage.SampleData
			if num, e := strconv.Atoi(params["last"]); e == nil {
				switch params["type"] {
				case "sample":
					s = &storage.SerieSample{Stag: label, Sts: support.Timestamp()}
				case "entry":
					s = &storage.SerieEntries{Stag: label, Sts: support.Timestamp()}
				default:
					_, _ = fmt.Fprintf(w, "")
					return
				}
				if tag, ts, vals, e := storage.ReadLastNTS(s, num, params["analysis"] != "current"); e == nil {
					rt := s.UnmarshalSliceSS(tag, ts, vals)
					_ = json.NewEncoder(w).Encode(rt)
				}
			}
		} else {
			if params["start"] != "" && params["end"] != "" {
				var st, en int64
				var e error
				if st, e = strconv.ParseInt(params["start"], 10, 64); e != nil {
					_, _ = fmt.Fprintf(w, "")
					return
				}
				if en, e = strconv.ParseInt(params["end"], 10, 64); e != nil {
					_, _ = fmt.Fprintf(w, "")
					return
				}
				if st >= en {
					_, _ = fmt.Fprintf(w, "")
					return
				}
				var s0, s1 storage.SampleData
				switch params["type"] {
				case "sample":
					s0 = &storage.SerieSample{Stag: label, Sts: st}
					s1 = &storage.SerieSample{Stag: label, Sts: en}
				case "entry":
					//if authorised {
					s0 = &storage.SerieEntries{Stag: label, Sts: st}
					s1 = &storage.SerieEntries{Stag: label, Sts: en}
					//}
				default:
					_, _ = fmt.Fprintf(w, "")
					return
				}
				//fmt.Println(st,en)

				var rt []storage.SampleData
				if tag, ts, vals, e := storage.ReadSeriesTS(s0, s1, params["analysis"] != "current"); e == nil {
					rt = s0.UnmarshalSliceSS(tag, ts, vals)
				}
				// the if is added to deal with timeout issues due to the fact this read can eb too long
				if e = json.NewEncoder(w).Encode(rt); e != nil {
					_, _ = fmt.Fprintf(w, "")
				}
			}
		}
	})
}
