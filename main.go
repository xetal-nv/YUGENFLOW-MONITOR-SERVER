package main

import (
	"context"
	"countingserver/servers"
	"countingserver/spaces"
	"countingserver/support"
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

const logfilename string = "logfile"

// TODO add TCP to start server and remove call to servers.StartTCP
func main() {
	defer support.CloseLog()
	if e := godotenv.Load(); e != nil {
		fmt.Println(e)
	} else {
		if os.Getenv("DELLOG") == "1" {
			//noinspection GoUnhandledErrorResult
			os.Remove(logfilename)
		}
		support.SetUpLog(logfilename)

		//servers.StartServers()

		// the part below needs to go to servers.StartServers()
		spaces.SetUp()
		spaces.CountersSetpUp()
		servers.SetUpTCP()
		servers.StartTCP(make(chan context.Context))
	}
}
