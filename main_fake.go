package main

import (
	"countingserver/gates"
	"countingserver/spaces"
	"countingserver/storage"
	"countingserver/support"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
	"time"
)

type Command struct {
	Valid    bool   `json:"valid"`
	Typedata string `json:"typedata"`
	Space    string `json:"Space"`
	Analisys string `json:"analisys"`
}

func referenceHandler3Ways(path string) http.Handler {
	cp := strings.Split(strings.Trim(path, "/"), "/")
	rt := Command{true, "", "", ""}

	if len(cp) != 3 {
		rt.Valid = false
	} else {
		rt.Typedata = cp[0]
		rt.Space = cp[1]
		rt.Analisys = cp[2]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("servers.referenceHandler3Ways: recovering from: ", e)
				}
			}
		}()
		//noinspection GoUnhandledErrorResult
		//fmt.Fprintf(w, "%s %s %s \n", r.Method, r.URL, r.Proto)
		fmt.Printf("%s %s %s \n", r.Method, r.URL, r.Proto)
		//Iterate over all header fields
		//for k, v := range r.Header {
		//	fmt.Fprintf(w, "Header field %q, Value %q\n", k, v)
		//}
		//
		//fmt.Fprintf(w, "Host = %q\n", r.Host)
		//fmt.Fprintf(w, "RemoteAddr= %q\n", r.RemoteAddr)
		////Get value for a specified token
		//fmt.Fprintf(w, "\n\nFinding value of \"Accept\" %q", r.Header["Accept"])

		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		json.NewEncoder(w).Encode(rt.Valid)

	})
}

// catches all unserved paths
func catchRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	json.NewEncoder(w).Encode(false)
}

func main_fake() {
	support.SupportSetUp("")

	// Set-up databases
	if err := storage.TimedIntDBSSetUp(false); err != nil {
		log.Fatal(err)
	}
	defer storage.TimedIntDBSClose()

	gates.SetUp()
	spaces.SetUp()
	support.SupportTerminate()

	mx := mux.NewRouter()

	server := &http.Server{
		Addr:           ":8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        mx,
	}

	//datatypes := []string{"sample", "entry"}
	//spaces := []string{"living", "studio", "bedroom"}
	//analisys := []string{"current", "hour", "day", "week"}
	//
	//for _, dt := range datatypes {
	//	for _, sp := range spaces {
	//		for _, ays := range analisys {
	//			path := "/" + strings.Join([]string{dt, sp, ays}, "/") + "/"
	//			mx.Handle(path, referenceHandler3Ways(path))
	//		}
	//	}
	//}

	handles := make(map[string]http.Handler)

	for dtn, dt := range spaces.LatestBankOut {
		for spn, sp := range dt {
			for alsn, _ := range sp {
				path := "/" + strings.Join([]string{strings.Trim(dtn, "_"), strings.Trim(spn, "_"),
					strings.Trim(alsn, "_")}, "/")
				fmt.Println("Serving", path)
				handles[path] = referenceHandler3Ways(path)
				//handles[path] = servers.registerHTTPhandles(path)
			}
		}
	}

	for p, h := range handles {
		mx.Handle(p, h)
	}

	mx.PathPrefix("/").Handler(http.HandlerFunc(catchRest))
	_ = server.ListenAndServe()
}
