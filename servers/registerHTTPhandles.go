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

type Answer struct {
	Valid bool                 `json:"valid"`
	Error string               `json:"errorcode"`
	Data  *storage.SerieSample `json:"data"`
}

func registerHTTPhandles(path string) http.Handler {

	sp := strings.Split(strings.Trim(path, "/"), "/")
	for i := range sp {
		sp[i] = support.StringLimit(sp[i], support.LabelLength)
	}
	fmt.Println(sp)

	rt := Answer{true, "", new(storage.SerieSample)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("servers.registerHTTPhandles: recovering from: ", e)
				}
			}
		}()
		fmt.Printf("%s %s %s \n", r.Method, r.URL, r.Proto)

		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		select {
		case data := <-spaces.LatestBankOut[sp[0]][sp[1]][sp[2]]:
			if e := rt.Data.Extract(data); e != nil {
				rt.Valid = false
				rt.Error = "ID"
			}
		case <-time.After(2000 * time.Millisecond):
			rt.Valid = false
			rt.Error = "TO"
		}

		//noinspection GoUnhandledErrorResult
		json.NewEncoder(w).Encode(rt)
	})
}
