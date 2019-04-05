package support

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"sync"
)

var Debug int
var LabelLength int

const logfilename string = "logfile" // logfile name
const TimeLayout = "15:04"           // time layout used to read the configuration file
const version = "v. 1.0.0"           // version

var CleanupLock = &sync.RWMutex{} // used to make sure clean-up on temrination does not affect critical operations

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
	if os.Getenv("DELLOG") == "1" {
		//noinspection GoUnhandledErrorResult
		os.Remove(logfilename)
	}
	LabelLength = 8
	if ll := os.Getenv("LENLABEL"); ll != "" {
		if v, e := strconv.Atoi(ll); e == nil {
			LabelLength = v
		}
	}

	if db := os.Getenv("DEBUGMODE"); db == "0" {
		Debug = 0
	} else {
		if v, e := strconv.Atoi(db); e == nil {
			Debug = v
		} else {
			panic("Fatal error:" + e.Error())
		}
	}
	// Set-up loggers
	if Debug == 0 {
		setUpLog(logfilename)
	}
	setUpDevLogger()
	log.Printf("Xetal Gate Server version: %v\n", version)
	log.Println("Maximum label length set to", LabelLength)
	if Debug != 0 {
		log.Printf("DEBUG MODE %v ACTIVE\n", Debug)
	}
}

func SupportTerminate() {
	closeLog()
}
