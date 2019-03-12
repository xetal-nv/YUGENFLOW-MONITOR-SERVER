package servers

import (
	"countingserver/support"
	"fmt"
	"log"
	"net/http"
)

//var counter int
//var once sync.Once

// TODO
// returns the DevLog
func dvlHTTHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				go func() {
					support.DLog <- support.DevData{"dvlHTTHandler",
						support.Timestamp(), "", []int{1}, true}
				}()
				log.Println("dvlHTTHandler: recovering from: ", e)
				//noinspection GoUnhandledErrorResult
				fmt.Fprintf(w, "Good try :-)")
			}
		}()
		//noinspection GoUnhandledErrorResult
		support.DLog <- support.DevData{Tag: "read"}
		_, _ = fmt.Fprintf(w, <-support.ODLog)
	})
}
