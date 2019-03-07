package main

import (
	"countingserver/gates"
	"countingserver/sensormodels"
	"countingserver/servers"
	"countingserver/spaces"
	"countingserver/storage"
	"countingserver/support"
	"log"
	"os"
)

func main() {
	defer support.SupportTerminate()
	support.SupportSetUp("")

	// Set-up databases
	if err := storage.TimedIntDBSSetUp(false); err != nil {
		log.Fatal(err)
	}
	defer storage.TimedIntDBSClose()

	gates.SetUp()
	spaces.SetUp()

	// testing
	switch os.Getenv("DEVMODE") {
	case "1":
		go sensormodels.Randgen()
	default:
	}

	// comment below for TCP debug
	// Set-up and start servers
	servers.StartServers()

	// Uncomment below for TCP debug
	//gates.SetUp()
	//spaces.SetUp()
	//servers.StartTCP(make(chan context.Context))

	// for the API use the globals SpaceDef and EntryList to extract the entire installation logical structure

}
