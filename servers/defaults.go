package servers

import (
	"context"
	"countingserver/spaces"
	"log"
	"net/http"
	"os"
	"strings"
)

const SIZE int = 2

var addServer [SIZE]string                  // server addresses
var sdServer [SIZE + 1]chan context.Context // channel for closure of servers
var hMap [SIZE]map[string]http.Handler      // server handler maps
var crcUsed bool                            // CRC used flag
var cmdchan chan []byte                     // channel to handler for receiving gate answers answer
var cmdlen map[byte]int                     // provides length for legal server2gate commands

func setupHTTP() error {

	hMap[0] = map[string]http.Handler{
		"/welcome": tempHTTPfuncHandler("Welcome to Go Web Development"),
		"/message": tempHTTPfuncHandler("net/http is awesome"),
		"/panic":   tempHTTPfuncHandler(""),
	}

	//hMap[1] = map[string]http.Handler{
	//	"/actuals": getCurrentSampleAPI(),
	//}

	hMap[1] = make(map[string]http.Handler)

	for dtn, dt := range spaces.LatestBankOut {
		for spn, sp := range dt {
			for alsn := range sp {
				path := "/" + strings.Join([]string{strings.Trim(dtn, "_"), strings.Trim(spn, "_"),
					strings.Trim(alsn, "_")}, "/")
				log.Println("ServersSetup: Serving API", path)
				hMap[1][path] = registerHTTPhandles(path)
			}
		}
	}

	for i, v := range strings.Split(os.Getenv("HTTPSPORTS"), ",") {
		addServer[i] = "0.0.0.0:" + strings.Trim(v, " ")
	}

	if addServer[0] == addServer[1] || addServer[0] == "" || addServer[1] == "" {
		log.Fatal("ServersSetup: fatal error: invalid addresses")
	}

	return nil
}
