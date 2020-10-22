package webService

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"gopkg.in/ini.v1"
	"log"
	"net/http"
	"os"
	"time"
)

func Start(sd chan bool) {

	var err error

	if webLog, err = mlogger.DeclareLog("yugenflow_webService", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_webService logfile.")
		os.Exit(0)
	}
	if e := mlogger.SetTextLimit(webLog, 50, 50, 12); e != nil {
		fmt.Println(e)
		os.Exit(0)
	}

	mlogger.Info(webLog,
		mlogger.LoggerData{"webService.Start",
			"service started",
			[]int{0}, true})

	if !globals.AccessData.Section("webservice").Key("disable").MustBool(false) {
		internalConfig, err := ini.InsensitiveLoad("webservice.ini")
		if err != nil {
			fmt.Printf("Fail to read webservice.ini file: %v", err)
			os.Exit(1)
		}

		apiLocation = internalConfig.Section("api").Key("location").MustString("")
		apiPort = internalConfig.Section("api").Key("port").MustString("8079")
		webServicePort = ":" + internalConfig.Section("service").Key("port").MustString("8095")

		mx := mux.NewRouter()
		server := &http.Server{
			Addr:           webServicePort,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        mx,
		}

		mx.Handle("/", http.FileServer(http.Dir("./webservice/public")))
		go func() {
			fmt.Printf("\nYugenFlow Web Service active on port %v\n\n", webServicePort)
			// always returns error. ErrServerClosed on graceful close
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				// unexpected error. port in use?
				log.Fatalf("webService failed to start: %v\n", err)
			}
		}()
		// shutdown
		<-sd
		_ = server.Shutdown(context.TODO())
	} else {
		fmt.Printf("*** WARNING: WebService not active ***\n")
		// shutdown
		<-sd
	}
	fmt.Println("Closing webService")
	mlogger.Info(webLog,
		mlogger.LoggerData{"webService.Start",
			"service stopped",
			[]int{0}, true})
	time.Sleep(time.Duration(settleTime) * time.Second)
	sd <- true
}
