package servers

import (
	"countingserver/support"
	"fmt"
	"log"
	"net/http"
)

func getCurrentSampleAPI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				support.DLog <- support.DevData{"servers.getCurrentSampleAPI recover",
					support.Timestamp(), "", []int{1}, true}
				log.Println("servers.getCurrentSampleAPI: recovering from: ", e)
			}
		}()
		//noinspection GoUnhandledErrorResult
		fmt.Fprintf(w, "%s %s %s \n", r.Method, r.URL, r.Proto)
		//Iterate over all header fields
		for k, v := range r.Header {
			fmt.Fprintf(w, "Header field %q, Value %q\n", k, v)
		}

		fmt.Fprintf(w, "Host = %q\n", r.Host)
		fmt.Fprintf(w, "RemoteAddr= %q\n", r.RemoteAddr)
		//Get value for a specified token
		fmt.Fprintf(w, "\n\nFinding value of \"Accept\" %q", r.Header["Accept"])
	})
}
