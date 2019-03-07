package servers

import (
	"context"
	"countingserver/support"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

func startHTTP(add string, sd chan context.Context, mh map[string]http.Handler) {

	mx := mux.NewRouter()

	server := &http.Server{
		Addr:           add,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        mx,
	}

	r := func() {
		ctx := <-sd
		//noinspection GoUnhandledErrorResult
		server.Shutdown(ctx)
	}

	go support.RunWithRecovery(r, nil)

	go func() {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("startHTTP: recovering server ", add, " from:\n ", e)
					sd <- context.Background() // close running shutdown goroutine
					startHTTP(add, sd, mh)
				}
			}
		}()

		stc := ""
		for p, h := range mh {
			if h != nil {
				mx.Handle(p, h)
			} else {
				stc = p
			}
		}
		if stc != "" {
			mx.PathPrefix("/").Handler(http.FileServer(http.Dir(stc)))
		}

		log.Println("startHTTP: Listening on server server: ", add)
		log.Panic("startHTTP: serve error: ", server.ListenAndServe())
	}()
}
