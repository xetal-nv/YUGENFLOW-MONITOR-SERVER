package main

import (
	"context"
	"countingserver/gates"
	"countingserver/registers"
	"countingserver/servers"
	"countingserver/spaces"
	"countingserver/support"
	"fmt"
	"github.com/joho/godotenv"
	"log"
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
		// Set-up loggers
		support.SetUpLog(logfilename)
		support.SetUpDevLogger()

		// Set-up databases
		if err := registers.TimedIntDBSSetUp(); err != nil {
			log.Fatal(err)
		}
		defer registers.TimedIntDBSClose()

		// Set-up and start servers
		//servers.StartServers()

		// the part below needs to go to servers.StartServers()
		gates.SetUp()
		spaces.SetUp()
		servers.StartTCP(make(chan context.Context))
	}
}
