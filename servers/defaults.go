package servers

import (
	"context"
	"countingserver/spaces"
	"countingserver/storage"
	"log"
	"net/http"
	"os"
	"strings"
)

const SIZE int = 2

type datafunc func() GenericData

var addServer [SIZE]string                  // server addresses
var sdServer [SIZE + 1]chan context.Context // channel for closure of servers
var hMap [SIZE]map[string]http.Handler      // server handler maps
var crcUsed bool                            // CRC used flag
var cmdBuffLen int                          // length of buffer for command channels
var sensorMac map[int][]byte                // map of sensor id to sensor MAC as provided by the sensor itself
var sensorChan map[int]chan []byte          // channel to handler managing commands to each connected sensor
var SensorCmd map[int]chan []byte           // externally visible channel to handler managing commands to each connected sensor
var dataMap map[string]datafunc             // used for HTTP API handling of different data types
var cmdAnswerLen = map[byte]int{            // provides length for legal server2gate commands
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

func setupHTTP() error {

	dataMap = make(map[string]datafunc)
	dataMap["sample"] = func() GenericData { return new(storage.SerieSample) }
	dataMap["entry"] = func() GenericData { return new(storage.SerieEntries) }

	hMap[0] = map[string]http.Handler{
		//"/welcome": tempHTTPfuncHandler("Welcome to Go Web Development"),
		//"/message": tempHTTPfuncHandler("net/http is awesome"),
		//"/panic":   tempHTTPfuncHandler(""),
		"./html/": nil,
	}

	//hMap[1] = map[string]http.Handler{
	//	"/actuals": getCurrentSampleAPI(),
	//}

	hMap[1] = make(map[string]http.Handler)

	hMap[1]["/dvl"] = dvlHTTHandler()

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

	for i, v := range strings.Split(os.Getenv("HTTPSPORTS"), ",") {
		addServer[i] = "0.0.0.0:" + strings.Trim(v, " ")
	}

	if addServer[0] == addServer[1] || addServer[0] == "" || addServer[1] == "" {
		log.Fatal("ServersSetup: fatal error: invalid addresses")
	}

	return nil
}
