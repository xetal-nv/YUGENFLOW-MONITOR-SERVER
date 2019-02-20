package main

import (
	"countingserver/registers"
	"countingserver/servers"
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

	// comment below for TCP debug
	// Set-up and start servers
	servers.StartServers()

	// Uncomment below for TCP debug
	//gates.SetUp()
	//spaces.SetUp()
	//servers.StartTCP(make(chan context.Context))
}
