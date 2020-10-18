package apiManager

import (
	"context"
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

func ApiManager(rst chan bool) {
	r := mux.NewRouter()
	r.Handle("/info", info())
	r.Handle("/connected", connectedSensors())
	r.Handle("/invalid", invalidSensors())
	r.Handle("/measurements", measurementDefinitions())
	r.Handle("/latestdata/{space}", latestData(false, false, 0))
	r.Handle("/latestdata", latestData(true, false, 0))
	r.Handle("/reference/{space}", latestData(false, true, 2))
	r.Handle("/reference", latestData(true, true, 2))
	r.Handle("/real/{space}", latestData(false, true, 1))
	r.Handle("/real", latestData(true, true, 1))

	//r.Handle("/register/{id}", register())
	//r.Handle("/dropdevice/{id}", deviceCommandLink("resetIdentifier"))
	//r.Handle("/yugenface/{id}/{cmd}", executeLink())
	//r.Handle("/settings/{id}", deviceCommand("configuration"))
	//r.Handle("/restart/{id}", deviceCommandLink("reset"))
	//r.Handle("/isolate/{id}", deviceCommandLink("block"))
	//r.Handle("/latestdata/{id}/{howmany}", deviceCommand("result"))
	//r.Handle("/operationmode/{id}/{mode}", deviceCommandLink("mode"))
	//r.Handle("/modellingstyle/{id}/{mode}", deviceCommandLink("localadaptation"))

	http.Handle("/", r)

	srv := &http.Server{
		Addr: "0.0.0.0:" + globals.APIport,
		// Good practice to set timeouts to avoid attacks.
		WriteTimeout: time.Second * time.Duration(globals.ServerTimeout),
		ReadTimeout:  time.Second * time.Duration(globals.ServerTimeout),
		IdleTimeout:  time.Second * 3 * time.Duration(globals.ServerTimeout),
		Handler:      r, // Pass our instance of gorilla/mux in.
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
		}
	}()

	mlogger.Info(globals.ClientManagerLog,
		mlogger.LoggerData{"apiManager.Start",
			"service started",
			[]int{1}, true})

	// setting up closure and shutdown
	<-rst
	fmt.Println("Closing ApiManager")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(globals.ServerTimeout))
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	_ = srv.Shutdown(ctx)
	rst <- true

}
