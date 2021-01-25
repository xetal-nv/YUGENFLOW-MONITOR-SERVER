// +build dev debug

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
	r.Handle("/latestdata/{space}", latestData(false, false, false, 0))
	r.Handle("/latestdata", latestData(true, false, false, 0))
	if !globals.DisableDatabase && globals.DBOPS {
		r.Handle("/reference/{space}", latestData(false, true, false, 2))
		r.Handle("/reference", latestData(true, true, false, 2))
		r.Handle("/delta/{space}", latestData(false, true, false, 1))
		r.Handle("/delta", latestData(true, true, false, 1))
		r.Handle("/series/delta/{space}", latestData(false, false, true, 1))
		r.Handle("/series/delta", latestData(true, false, true, 1))
		r.Handle("/series/reference/{space}", latestData(false, false, true, 2))
		r.Handle("/series/reference", latestData(true, false, true, 2))
		r.Handle("/presence/{space}", latestData(false, false, true, 3))
		r.Handle("/presence", latestData(true, false, true, 3))
	}
	r.Handle("/command/{command}", command())
	r.Handle("/devicedefinitions", devicedefinitions())
	r.Handle("/disconnect", disconnectDevice())

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

	mlogger.Info(globals.ApiManagerLog,
		mlogger.LoggerData{"apiManager.Start",
			"service started",
			[]int{1}, true})

	// setting up closure and shutdown
	<-rst
	fmt.Println("Closing apiManager.ApiManagerLog")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(globals.ServerTimeout))
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	_ = srv.Shutdown(ctx)
	rst <- true

}
