package support

import (
	"log"
	"os"
	"sync"
)

// set-ups the official log file

var logf *os.File
var e error
var once sync.Once

func setUpLog(n string) {
	once.Do(func() {
		if logf, e = os.OpenFile(n, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); e != nil {
			log.Fatal(e)
		}
		log.SetOutput(logf)
	})
}

func closeLog() {
	if logf != nil {
		_ = logf.Close()
	}
}
