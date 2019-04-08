package servers

import (
	"context"
	"gateserver/spaces"
	"gateserver/support"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const SIZE int = 2

type datafunc func() GenericData

var addServer [SIZE]string                  // server addresses
var sdServer [SIZE + 1]chan context.Context // channel for closure of servers
var hMap [SIZE]map[string]http.Handler      // server handler maps
var crcUsed bool                            // CRC used flag
//var cmdBuffLen int                          // length of buffer for command channels
var mutexSensorMaos = &sync.Mutex{} // this mutex is used to avoid concurrent writes at start-up on sensorMac, sensorMac,SensorCmd
var sensorMac map[int][]byte        // map of sensor id to sensor MAC as provided by the sensor itself
var sensorChan map[int]chan []byte  // channel to handler managing commands to each connected sensor
var SensorCmd map[int]chan []byte   // externally visible channel to handler managing commands to each connected sensor
var dataMap map[string]datafunc     // used for HTTP API handling of different data types
var cmdAnswerLen = map[byte]int{    // provides length for legal server2gate commands
	2:  1,
	3:  1,
	4:  1,
	5:  1,
	6:  1,
	8:  1,
	10: 1,
	12: 1,
	13: 1,
	14: 1,
	7:  3,
	9:  3,
	11: 3,
}
var timeout int
var resetbg struct {
	start    time.Time
	end      time.Time
	interval time.Duration
	valid    bool
}

func setJSenvironment() {
	if dat, e := ioutil.ReadFile("dbs/dat"); e == nil {
		f, err := os.Create("./html/js/dat.js")
		if err != nil {
			log.Fatal("Fatal error creating ip.js: ", err)
		}
		js := "var StartDat = " + string(dat) + ";"
		if _, err := f.WriteString(js); err != nil {
			_ = f.Close()
			log.Fatal("Fatal error writing to dat.js: ", err)
		}
		if err = f.Close(); err != nil {
			log.Fatal("Fatal error closing dat.js: ", err)
		}
	} else {
		log.Fatal("servers.setJSenvironment: fatal error cannot retrieve dbs/dat")
	}

	ports := strings.Split(os.Getenv("HTTPSPORTS"), ",")
	for i, v := range ports {
		if port := strings.Trim(v, " "); port != "" {
			addServer[i] = "0.0.0.0:" + strings.Trim(v, " ")
		} else {
			log.Fatal("ServersSetup: fatal error: invalid addresses")
		}
		for j, c := range addServer {
			if addServer[i] == c && i != j {
				log.Fatal("ServersSetup: fatal error: invalid addresses")
			}
		}
	}
	ip := ""
	if ip = os.Getenv("IP"); ip == "" {
		ip = support.GetOutboundIP().String()
	}

	f, err := os.Create("./html/js/ip.js")
	if err != nil {
		log.Fatal("Fatal error creating ip.js: ", err)
	}
	js := "var ip = \"http://" + ip + ":" + strings.Trim(ports[len(ports)-1], " ") + "\";"
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to ip.js: ", err)
	}
	if err = f.Close(); err != nil {
		log.Fatal("Fatal error closing ip.js: ", err)
	}
	f, err = os.Create("./html/js/sw.js")
	if err != nil {
		log.Fatal("Fatal error creating sw.js: ", err)
	}
	js = "var samplingWindow = " + strconv.Itoa(spaces.SamplingWindow) + " * 1000;"
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to sw.js: ", err)
	}
	if err = f.Close(); err != nil {
		log.Fatal("Fatal error closing sw.js: ", err)
	}
}
