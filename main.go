package main

import (
	"context"
	"countingserver/gates"
	"countingserver/registers"
	"countingserver/servers"
	"countingserver/spaces"
	"countingserver/support"
	"log"
)

// TODO add TCP to start server and remove call to servers.StartTCP
func main() {
	defer support.SupportTerminate()
	support.SupportSetUp("")

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
