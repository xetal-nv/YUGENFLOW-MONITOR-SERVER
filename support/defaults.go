package support

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
)

var Debug int

const logfilename string = "logfile"

func SupportSetUp(envf string) {
	if e := godotenv.Load(envf); e != nil {
		panic("Fatal error:" + e.Error())
	} else {
		if os.Getenv("DELLOG") == "1" {
			//noinspection GoUnhandledErrorResult
			os.Remove(logfilename)
		}
		if db := os.Getenv("DEBUGMODE"); db == "0" {
			Debug = 0
			fmt.Println("DEBUG MODE ACTIVE")
		} else {
			if v, e := strconv.Atoi(db); e == nil {
				Debug = v
			} else {
				panic("Fatal error:" + e.Error())
			}
		}
		// Set-up loggers
		if Debug > 0 {
			setUpLog(logfilename)
		}
		setUpDevLogger()
	}
}

func SupportTerminate() {
	closeLog()
}
