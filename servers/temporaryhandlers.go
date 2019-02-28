package servers

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
)

//var counter int
//var once sync.Once

// test Handler
func tempHTTPfuncHandler(message string) http.Handler {
	m := message
	log.Println("Test Handler: started")
	if rand.Intn(5) == 2 {
		panic("setupHTTP error")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				if e != nil {
					log.Println("Test tempHTTPfuncHandler: recovering from: ", e)
					//noinspection GoUnhandledErrorResult
					fmt.Fprintf(w, "Good try :-)")
				}
			}
		}()
		//noinspection GoUnhandledErrorResult
		fmt.Fprintf(w, m)
		if m == "" {
			panic("panic address")
		}
	})
}
