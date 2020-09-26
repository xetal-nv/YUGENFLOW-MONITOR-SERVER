package sensorManager

import (
	"fmt"
	"gateserver/dbs/sensorDB"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"strconv"
	"sync"
	"time"
	"xetal.ddns.net/utils/recovery"
)

var once sync.Once
var setIdCh chan interface{}

func maliciousSetIdDOS(ipc, mac string) bool {
	once.Do(func() {
		setIdCh = make(chan interface{}, globals.SecurityLength)
		go recovery.RunWith(
			func() {
				others.ChannelEmptier(setIdCh, make(chan bool, 1), globals.RepetitiveTimeout)
			},
			nil)
		if globals.DebugActive {
			fmt.Printf("*** WARNING: setID DoS check started %v:%v ***\n", globals.SecurityLength, globals.RepetitiveTimeout)
		}
		mlogger.Info(globals.SensorManagerLog,
			mlogger.LoggerData{"sensorManager.maliciousSetIdDOS",
				"service started " + strconv.Itoa(globals.SecurityLength) + ":" + strconv.Itoa(globals.RepetitiveTimeout),
				[]int{0}, true})
	})
	select {
	case setIdCh <- nil:
		return false
	case <-time.After(time.Duration(globals.RepetitiveTimeout/10) * time.Second):
		_, _ = sensorDB.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
		_, _ = sensorDB.MarkMAC([]byte(mac), globals.MaliciousTriesMac)
		return true
	}
}
