package servers

import (
	"context"
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

	go func() {
		ctx := <-sd
		//noinspection GoUnhandledErrorResult
		server.Shutdown(ctx)
	}()

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

		for p, h := range mh {
			mx.Handle(p, h)
		}

		log.Println("startHTTP: Listening on server server: ", add)
		log.Panic("startHTTP: serve error: ", server.ListenAndServe())
	}()
}
