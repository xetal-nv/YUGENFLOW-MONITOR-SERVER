package support

import (
	"log"
	"os"
	"sync"
)

var logf *os.File
var e error
var o1, o2 sync.Once

func SetUpLog(n string) {
	o1.Do(func() {
		if logf, e = os.OpenFile(n, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); e != nil {
			log.Fatal(e)
		}
		log.SetOutput(logf)
	})
}

func CloseLog() {
	if logf != nil {
		_ = logf.Close()
	}
}
