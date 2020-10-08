package support

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var Debug int
var LabelLength int
var DelLogs = false
var MalOn = true
var RstON = false

const logFileName string = "gnl" // logfile name
const TimeLayout = "15:04"       // time layout used to read the configuration file

const SkipDBS = true                   // if true operations towards the DBS will be skipped
const DisableWebApp = false || SkipDBS // if true web app will be disabled

var CleanupLock = &sync.RWMutex{} // used to make sure clean-up on termination does not affect critical operations

// set-ups all support variables according to the configuration file .env

func SetUp(envf string) {
	if envf == "" {
		if _, err := os.Stat(".systemenv"); err == nil {
			if e := godotenv.Load(".systemenv"); e != nil {
				panic("Fatal error:" + e.Error())
			}
		} else {
			if e := godotenv.Load(); e != nil {
				panic("Fatal error:" + e.Error())
			}
		}
	} else {
		if _, err := os.Stat(".systemenv"); err == nil {
			if e := godotenv.Load(".systemenv", envf); e != nil {
				panic("Fatal error:" + e.Error())
			}
		} else {
			if e := godotenv.Load(envf); e != nil {
				panic("Fatal error:" + e.Error())
			}
		}
		//if e := godotenv.Load(envf); e != nil {
		//	panic("Fatal error:" + e.Error())
		//}
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
		go setUpLog(logFileName, time.Now().Local(), c)
		<-c
	} else {
		pwd, _ := os.Getwd()
		if DelLogs {
			_ = os.RemoveAll(filepath.Join(pwd, "log"))
		}
		_ = os.MkdirAll("log", os.ModePerm)
	}
	setUpDevLogger()
	log.Println("Starting server ...")
	log.Println("Maximum label length set to", LabelLength)
}

func Terminate() {
	closeLog()
}
