package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

type User struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Age       int    `json:"age"`
}

func getCurrentSampleAPI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("servers.getCurrentSampleAPI: recovering from: ", e)
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

		peter := User{
			Firstname: "John",
			Lastname:  "Doe",
			Age:       25,
		}

		json.NewEncoder(w).Encode(peter)

	})
}

func main_test() {

	mx := mux.NewRouter()

	server := &http.Server{
		Addr:           ":8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        mx,
	}

	mx.Handle("/", getCurrentSampleAPI())
	server.ListenAndServe()
}
