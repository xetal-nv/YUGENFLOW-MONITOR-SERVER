package support

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// set-ups the official log file

var RotInt = 360
var RotSize int64 = 10000000
var logf *os.File
var e error

func closeLog() {
	//if logf != nil {
	_ = logf.Close()
	//}
}

func setUpLog(n string, ct time.Time, c chan bool) {

	defer func() {
		if e := recover(); e != nil {
			go func() {
				DLog <- DevData{"support.setUpRotatingLog: recovering server", Timestamp(), "", []int{1}, true}
			}()
			setUpLog(n, ct, nil)
		}
	}()

	pwd, _ := os.Getwd()
	if DelLogs {
		_ = os.RemoveAll(filepath.Join(pwd, "log"))
	}
	_ = os.MkdirAll("log", os.ModePerm)
	rf := filepath.Join(pwd, "log", n+"_"+ct.Format("2006-01-02"))

	// look for latest lof file
	found := true
	var ind = 0
	for found {
		file := rf + "_" + strconv.Itoa(ind) + ".log"
		if _, err := os.Stat(file); err == nil {
			ind += 1
		} else if os.IsNotExist(err) {
			found = false
		} else {
			// in case of system errors we report it to console and change name
			ind = 0
			found = false
			rf += "_se"
		}
	}

	file := rf + "_" + strconv.Itoa(ind) + ".log"
	if logf, e = os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); e != nil {
		log.Fatal(e)
	}
	log.SetOutput(logf)

	// force sync for proper info at start
	if c != nil {
		go func() { c <- true }()
	}

	log.Printf("WARNING: setting logger ri and rs are set to %v and %v\n", RotInt, RotSize)

	for {
		time.Sleep(time.Duration(RotInt) * time.Minute)
		fi, err := os.Stat(file)
		if err == nil {
			if size := fi.Size(); size > RotSize {
				ind += 1
				file = rf + "_" + strconv.Itoa(ind) + ".log"
				if newLogFile, ne := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); ne == nil {
					log.SetOutput(newLogFile)
					_ = logf.Close()
					logf = newLogFile
				} else {
					log.Println("support.Logger: failed create new log file:", ind)
				}
			}
		}
	}
}
