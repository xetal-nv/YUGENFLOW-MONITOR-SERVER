package servers

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
)

const SIZE int = 2

var addServer [SIZE]string
var sdServer [SIZE]chan context.Context
var hMap [SIZE]map[string]http.Handler
var crcUsed bool

func setup() error {

	hMap[0] = map[string]http.Handler{
		"/welcome": tempHTTPfuncHandler("Welcome to Go Web Development"),
		"/message": tempHTTPfuncHandler("net/http is awesome"),
		"/panic":   tempHTTPfuncHandler(""),
	}

	hMap[1] = map[string]http.Handler{
		"/welcome": tempHTTPfuncHandler("Welcome to Go Web Development"),
		"/message": tempHTTPfuncHandler("net/http is awesome"),
		"/panic":   tempHTTPfuncHandler(""),
	}

	for i, v := range strings.Split(os.Getenv("HTTPSPORTS"), ",") {
		addServer[i] = "0.0.0.0:" + strings.Trim(v, " ")
	}

	if addServer[0] == addServer[1] || addServer[0] == "" || addServer[1] == "" {
		log.Fatal("ServersSetup: fatal error: invalid addresses")
	}

	return nil
}
