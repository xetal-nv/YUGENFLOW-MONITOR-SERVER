package servers

import (
	"encoding/json"
	"fmt"
	"gateserver/spaces"
	"gateserver/storage"
	"gateserver/supp"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
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
	var commands = []string{"last", "type", "space", "analysis", "start", "end"}

	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"servers.seriesHTTPhandler",
						supp.Timestamp(), "handle crashed", []int{1}, true}
				}()
				log.Println("servers.seriesHTTPhandler: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// check if the client is authorised to access debug data
		ip := strings.Split(strings.Replace(r.RemoteAddr, "[::1]", "localhost", 1), ":")[0]
		authorised := false
		dbgMutex.RLock()
		if tts, ok := dbgRegistry[ip]; ok {
			if (supp.Timestamp()-tts)/1000 <= (authDbgInterval * 60) {
				authorised = true
			} else {
				delete(dbgRegistry, ip)
			}
		}
		dbgMutex.RUnlock()
		// if authorised {
		// 	log.Println("servers.seriesHTTPhandler: answering (debug) authorised device at address:", ip)
		// }
		//fmt.Println(authorised, ip)

		params := make(map[string]string)
		for _, i := range commands {
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

		// fmt.Println(params)

		label := supp.StringLimit(params["type"], supp.LabelLength)
		label += supp.StringLimit(params["space"], supp.LabelLength)
		label += supp.StringLimit(params["analysis"], supp.LabelLength)

		//fmt.Println(label)
		//os.Exit(1)

		if params["last"] != "" {
			var s storage.SampleData
			if num, e := strconv.Atoi(params["last"]); e == nil {
				switch params["type"] {
				case "sample":
					s = &storage.SeriesSample{Stag: label, Sts: supp.Timestamp()}
				case "entry":
					s = &storage.SeriesEntries{Stag: label, Sts: supp.Timestamp()}
				default:
					_, _ = fmt.Fprintf(w, "")
					return
				}
				if tag, ts, vals, e := storage.ReadLastNTS(s, num, params["analysis"] != "current"); e == nil {
					rt := s.UnmarshalSliceSS(tag, ts, vals)
					if e := json.NewEncoder(w).Encode(rt); e != nil {
						_, _ = fmt.Fprintf(w, "")
					}
				}
			}
		} else {
			var rt []storage.SampleData
			if params["start"] != "" && params["end"] != "" {

				// if not in authorised via pin, the request will not provide data for the current day
				if !authorised {
					if eni, err := strconv.Atoi(params["end"]); err == nil {
						en := time.Unix(int64(eni/1000), 0)
						ts := time.Now()
						yearen, monthen, dayen := en.Date()
						yearts, monthts, dayts := ts.Date()
						if (yearen == yearts) && (monthen == monthts) && (dayen == dayts) {
							h, m, s := en.Clock()
							en = en.Add(-time.Duration(h)*time.Hour - time.Duration(m)*time.Minute - time.Duration(s)*time.Second)
							params["end"] = strconv.FormatInt(en.Unix(), 10) + "000"
						}
					} else {
						_, _ = fmt.Fprintf(w, "")
						return
					}
				} else {
					log.Println("servers.seriesHTTPhandler: answering (debug) authorised device at address:", ip)
				}

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
					s0 = &storage.SeriesSample{Stag: label, Sts: st}
					s1 = &storage.SeriesSample{Stag: label, Sts: en}
					// fmt.Println(s0, s1)
					if tag, ts, vals, e := storage.ReadSeriesTS(s0, s1, params["analysis"] != "current"); e == nil {
						rt = s0.UnmarshalSliceSS(tag, ts, vals)
					}
					// fmt.Println(rt)
					if e := json.NewEncoder(w).Encode(rt); e != nil {
						_, _ = fmt.Fprintf(w, "")
					}
				case "entry":
					// with entry we provide both samples and flows data, and check for inconsistencies
					var convertedRt []storage.JsonSeriesEntries
					var dataPeriod int
					var fullReport storage.JsonCompleteReport
					fullReport.Stag = label
					for _, el := range spaces.AvgAnalysis {
						if el.Name == supp.StringLimit(params["analysis"], supp.LabelLength) {
							dataPeriod = el.Interval
							fullReport.Meas = params["analysis"]
						}
					}

					s0s := &storage.SeriesSample{Stag: strings.Replace(label, "entry___", "sample__", -1), Sts: st}
					s1s := &storage.SeriesSample{Stag: strings.Replace(label, "entry___", "sample__", -1), Sts: en}
					var referenceSamples []storage.SeriesSample
					if tag, ts, vals, e := storage.ReadSeriesTS(s0s, s1s, params["analysis"] != "current"); e == nil {
						referenceSamples = s0s.UnmarshalSliceNative(tag, ts, vals)
					}

					s0 = &storage.SeriesEntries{Stag: label, Sts: st}
					s1 = &storage.SeriesEntries{Stag: label, Sts: en}
					if tag, ts, vals, e := storage.ReadSeriesTS(s0, s1, params["analysis"] != "current"); e == nil {
						rt = s0.UnmarshalSliceSS(tag, ts, vals)
					}
					for _, v := range rt {
						convertedTemp := storage.JsonSeriesEntries{}
						seVal := storage.SeriesEntries{}
						codedVal := v.Marshal()
						_ = seVal.Unmarshal(codedVal)
						seVal.Stag = v.Tag()
						convertedTemp.ExpandEntries(seVal)
						convertedRt = append(convertedRt, convertedTemp)
					}

					// merge data marking data without dual entry as corrupted
					// data received from the DBS is ordered in time with timestamps in ms
					iFlow := 0
					for _, el := range referenceSamples {
						ns := storage.JsonCompleteData{Sts: (el.Sts / 1000) * 1000, AvgPresence: el.Sval}
						if iFlow < len(convertedRt) {
							if int(math.Abs(float64(el.Sts-convertedRt[iFlow].Sts)/1000)) <= dataPeriod/2 {
								ns.Sval = convertedRt[iFlow].Sval
								iFlow += 1
							} else {
								// we have corrupted data
								for convertedRt[iFlow].Sts < el.Sts-int64(dataPeriod*1000/2) {
									tmp := storage.JsonCompleteData{Sts: (convertedRt[iFlow].Sts / 1000) * 1000,
										Corrupted: true, Sval: convertedRt[iFlow].Sval}
									fullReport.Data = append(fullReport.Data, tmp)
									iFlow = +1
									if iFlow >= len(convertedRt) {
										break
									}
								}
								if iFlow < len(convertedRt) {
									if int(math.Abs(float64(el.Sts-convertedRt[iFlow].Sts)/1000)) <= dataPeriod/2 {
										ns.Sval = convertedRt[iFlow].Sval
									} else {
										ns.Corrupted = true
									}
									iFlow = +1
								} else {
									// we have corrupted data
									ns.Corrupted = true
								}
							}
						} else {
							// we have corrupted data
							ns.Corrupted = true
						}
						fullReport.Data = append(fullReport.Data, ns)
					}
					// If we have corrupted flow data, we add it at the end
					for iFlow < len(convertedRt) {
						tmp := storage.JsonCompleteData{Sts: (convertedRt[iFlow].Sts / 1000 * 1000), Corrupted: true, Sval: convertedRt[iFlow].Sval}
						fullReport.Data = append(fullReport.Data, tmp)
						iFlow = +1
					}

					if e := json.NewEncoder(w).Encode(fullReport); e != nil {
						_, _ = fmt.Fprintf(w, "")
					}
				default:
					_, _ = fmt.Fprintf(w, "")
					return
				}
			} else {
				_, _ = fmt.Fprintf(w, "")
			}
		}
	})
}
