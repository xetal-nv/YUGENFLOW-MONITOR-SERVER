package servers

import (
	"fmt"
	"gateserver/storage"
	"gateserver/supp"
	"log"
	"net/http"
	"os"
)

// start database sample values retrieval following format given in .recoverysamples
// this includes also removal of all sample data in the given interval
func retrieveDBSsamples() http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"loadsamples",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("loadsamples: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		go storage.RetrieveSampleFromFile(false)

		_, _ = fmt.Fprintf(w, "Sample DBS retrieval initiated")
	})
}

// start database sample values retrieval following format given in .recoverypresence
// this includes also removal of all sample data in the given interval
func retrieveDBSpresence() http.Handler {
	cors := false
	if os.Getenv("CORS") != "" {
		cors = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					supp.DLog <- supp.DevData{"loadsamples",
						supp.Timestamp(), "", []int{1}, true}
				}()
				log.Println("loadsamples: recovering from: ", e)
			}
		}()

		//Allow CORS here By * or specific origin
		if cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		go storage.RetrievePresenceFromFile(false)

		_, _ = fmt.Fprintf(w, "Presence DBS retrieval initiated")
	})
}
