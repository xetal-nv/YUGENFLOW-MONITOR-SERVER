package servers

import (
	"encoding/json"
	"fmt"
	"gateserver/storage"
	"gateserver/supp"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func presenceHTTPhandler() http.Handler {
	var cmds = []string{"space", "analysis", "start", "end"}

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"servers,seriesHTTPhandler",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("servers.seriesHTTPhandler: recovering from: ", e)
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
					supp.DLog <- supp.DevData{"servers.seriesHTTPhandler: " + strings.Trim(val[0], " "),
						supp.Timestamp(), "illegal request", []int{1}, true}
				}()
				_, _ = fmt.Fprintf(w, "")
				return
			}
		}

		label := supp.StringLimit("presence", supp.LabelLength)
		label += supp.StringLimit(params["space"], supp.LabelLength)
		label += supp.StringLimit(params["analysis"], supp.LabelLength)

		//fmt.Println(label)

		if params["start"] != "" && params["end"] != "" {
			var st, en int64
			var e error
			if st, e = strconv.ParseInt(params["start"], 10, 64); e != nil {
				//fmt.Fprintf(w,"Error in start parameter")
				_, _ = fmt.Fprintf(w, "")
				return
			}
			if en, e = strconv.ParseInt(params["end"], 10, 64); e != nil {
				//fmt.Fprintf(w,"Error in end parameter")
				_, _ = fmt.Fprintf(w, "")
				return
			}
			if st >= en {
				//fmt.Fprintf(w,"Error as start parameter is later then the end one")
				_, _ = fmt.Fprintf(w, "")
				return
			}
			//fmt.Println(st,en)
			var s0, s1 storage.SampleData
			s0 = &storage.SeriesSample{Stag: label, Sts: st}
			s1 = &storage.SeriesSample{Stag: label, Sts: en}

			var rt []storage.SampleData
			if tag, ts, vals, e := storage.ReadSeriesSD(s0, s1, true); e == nil {
				//fmt.Println(tag, ts, vals)
				//for _, val := range ts {
				//	fmt.Println(time.Unix(val/1000, 0))
				//}
				rt = s0.UnmarshalSliceSS(tag, ts, vals)
			}
			// the if is added to deal with timeout issues due to the fact this read can eb too long
			if e = json.NewEncoder(w).Encode(rt); e != nil {
				_, _ = fmt.Fprintf(w, "")
			}
		}
	})
}
