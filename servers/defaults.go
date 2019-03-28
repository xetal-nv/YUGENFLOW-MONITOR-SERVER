package servers

import (
	"context"
	"countingserver/spaces"
	"countingserver/storage"
	"countingserver/support"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const SIZE int = 2

type datafunc func() GenericData

var addServer [SIZE]string                  // server addresses
var sdServer [SIZE + 1]chan context.Context // channel for closure of servers
var hMap [SIZE]map[string]http.Handler      // server handler maps
var crcUsed bool                            // CRC used flag
//var cmdBuffLen int                          // length of buffer for command channels
var sensorMac map[int][]byte       // map of sensor id to sensor MAC as provided by the sensor itself
var sensorChan map[int]chan []byte // channel to handler managing commands to each connected sensor
var SensorCmd map[int]chan []byte  // externally visible channel to handler managing commands to each connected sensor
var dataMap map[string]datafunc    // used for HTTP API handling of different data types
var cmdAnswerLen = map[byte]int{   // provides length for legal server2gate commands
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

// set-up of HTTP serverd and handlers
func setupHTTP() error {

	dataMap = make(map[string]datafunc)
	dataMap["sample"] = func() GenericData { return new(storage.SerieSample) }
	dataMap["entry"] = func() GenericData { return new(storage.SerieEntries) }

	// enable web server
	hMap[0] = map[string]http.Handler{
		"./html/": nil,
	}

	hMap[1] = make(map[string]http.Handler)
	// development log API
	hMap[1]["/dvl"] = dvlHTTHandler()
	// installation information API
	hMap[1]["/info"] = infoHTTHandler()
	// Series data retrieval API
	hMap[1]["/series"] = seriesHTTPhandler()
	// Sensor command API
	hMap[1]["/cmd"] = commandHTTHandler()
	// analysis information API
	hMap[1]["/asys"] = asysHTTHandler()

	// add SVG API for installation graphs
	for spn := range spaces.SpaceDef {
		name := strings.Replace(spn, "_", "", -1)
		hMap[1]["/plan/"+name] = planHTTPHandler(name)
	}
	hMap[1]["/plan/logo"] = planHTTPHandler("logo")

	// Real time data retrieval API
	for dtn, dt := range spaces.LatestBankOut {
		ref := strings.Trim(dtn, "_")
		keysSpaces := make(map[string][]string)
		for spn, sp := range dt {
			subpath := "/" + strings.Trim(dtn, "_") + "/" + strings.Trim(spn, "_")
			//log.Println("ServersSetup: Serving API", subpath)
			var keysType []string
			for alsn := range sp {
				path := subpath + "/" + strings.Trim(alsn, "_")
				keysType = append(keysType, alsn)

				if _, ok := dataMap[ref]; ok {
					log.Println("ServersSetup: Serving API", path)
					hMap[1][path] = singleRegisterHTTPhandler(path, ref)
				}
			}
			ref := strings.Trim(dtn, "_")
			if _, ok := dataMap[ref]; ok {
				log.Println("ServersSetup: Serving API", subpath)
				hMap[1][subpath] = spaceRegisterHTTPhandler(subpath, keysType, ref)
			}
			keysSpaces[spn] = keysType
		}
		p := "/" + strings.Trim(dtn, "_")
		log.Println("ServersSetup: Serving API", p)
		hMap[1][p] = datatypeRegisterHTTPhandler(p, keysSpaces)
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
	return nil
}
