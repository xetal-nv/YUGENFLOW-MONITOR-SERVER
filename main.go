package main

import (
	"countingserver/servers"
	"countingserver/storage"
	"countingserver/support"
	"log"
)

func main() {
	defer support.SupportTerminate()
	support.SupportSetUp("")

	// Set-up databases
	if err := storage.TimedIntDBSSetUp(); err != nil {
		log.Fatal(err)
	}
	defer storage.TimedIntDBSClose()

	// comment below for TCP debug
	// Set-up and start servers
	servers.StartServers()

	// Uncomment below for TCP debug
	//gates.SetUp()
	//spaces.SetUp()
	//servers.StartTCP(make(chan context.Context))
}
