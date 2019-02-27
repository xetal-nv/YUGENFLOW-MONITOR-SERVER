package support

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

var Debug int
var LabelLength int

const logfilename string = "logfile"

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
	log.Println("Maximum label length set to", LabelLength)
	if db := os.Getenv("DEBUGMODE"); db == "0" {
		Debug = 0
	} else {
		if v, e := strconv.Atoi(db); e == nil {
			Debug = v
			log.Printf("DEBUG MODE %v ACTIVE\n", v)
		} else {
			panic("Fatal error:" + e.Error())
		}
	}
	// Set-up loggers
	if Debug == 0 {
		setUpLog(logfilename)
	}
	setUpDevLogger()
}

func SupportTerminate() {
	closeLog()
}
