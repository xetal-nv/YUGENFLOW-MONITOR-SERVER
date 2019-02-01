package main

import (
	"context"
	"countingserver/servers"
	"countingserver/support"
	"fmt"
	"github.com/joho/godotenv"
)

const logfilename string = "logfile"

func main() {
	//_ = os.Remove(logfilename) // for testing only
	support.SetUpLog(logfilename)
	defer support.CloseLog()
	if e := godotenv.Load(); e != nil {
		fmt.Println(e)
	}
	//testnonblocking()
	//testtemplate()
	//servers.StartServers()
	//testrecover()
	servers.StartTCP(make(chan context.Context))
}
