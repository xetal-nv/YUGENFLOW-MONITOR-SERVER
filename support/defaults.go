package support

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

var Debug int
var LabelLength int
var Dellogs = false
var MalOn = true

const logfilename string = "gnl" // logfile name
const TimeLayout = "15:04"       // time layout used to read the configuration file

var CleanupLock = &sync.RWMutex{} // used to make sure clean-up on termination does not affect critical operations

// set-ups all support variables according to the configuration file .env

func SupportSetUp(envf string) {
	if envf == "" {
		if e := godotenv.Load(); e != nil {
			panic("Fatal error:" + e.Error())
		}
	} else {
		if e := godotenv.Load(envf); e != nil {
			panic("Fatal error:" + e.Error())
		}
	}

	LabelLength = 8
	if ll := os.Getenv("LENLABEL"); ll != "" {
		if v, e := strconv.Atoi(ll); e == nil {
			LabelLength = v
		}
	}

	// Set-up loggers
	if Debug == 0 {
		c := make(chan bool)
		go setUpLog(logfilename, time.Now().Local(), c)
		<-c
	}
	setUpDevLogger()
	log.Println("Starting server ...")
	log.Println("Maximum label length set to", LabelLength)
}

func SupportTerminate() {
	closeLog()
}
