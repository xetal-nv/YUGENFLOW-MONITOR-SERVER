package servers

import (
	"context"
	"gateserver/spaces"
	"gateserver/storage"
	"gateserver/support"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

var Dvl bool = false

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
	var rmode string
	if rmode = os.Getenv("RMODE"); rmode == "" {
		rmode = "0"
	}

	f, err = os.Create("./html/js/rmode.js")
	if err != nil {
		log.Fatal("Fatal error creating rmode.js: ", err)
	}
	js = "var rmode = " + rmode + ";"
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to rmode.js: ", err)
	}
	if err = f.Close(); err != nil {
		log.Fatal("Fatal error closing rmode.js: ", err)
	}
	log.Printf("Reporting mode set to %v\n", rmode)
}

// set-up of HTTP servers and handlers
func setupHTTP() error {

	setJSenvironment()

	dataMap = make(map[string]datafunc)
	dataMap["sample"] = func() GenericData { return new(storage.SerieSample) }
	dataMap["entry"] = func() GenericData { return new(storage.SerieEntries) }

	// enable web server
	hMap[0] = map[string]http.Handler{
		"./html/": nil,
	}

	hMap[1] = make(map[string]http.Handler)
	// development log API
	if Dvl {
		hMap[1]["/dvl"] = dvlHTTHandler()
		log.Println("WARNING: Developer Logfile enabled")
	}
	// installation information API
	hMap[1]["/info"] = infoHTTHandler()
	// Series data retrieval API
	hMap[1]["/series"] = seriesHTTPhandler()
	// Sensor command API
	hMap[1]["/cmd"] = commandHTTHandler()
	// analysis information API
	hMap[1]["/asys"] = asysHTTHandler()
	// unused registered device API
	hMap[1]["/und"] = unusedDeviceHTTPHandler()
	// pending registered device API
	hMap[1]["/pending"] = pendingDeviceHTTPHandler()
	// unknown registered device API and its variants
	hMap[1]["/udef"] = undefinedDeviceHTTPHandler("")
	hMap[1]["/udef/active"] = undefinedDeviceHTTPHandler("active")
	hMap[1]["/udef/defined"] = undefinedDeviceHTTPHandler("defined")
	hMap[1]["/udef/undefined"] = undefinedDeviceHTTPHandler("undefined")
	hMap[1]["/udef/notactive"] = undefinedDeviceHTTPHandler("notactive")
	// unused registered device API
	hMap[1]["/active"] = usedDeviceHTTPHandler()

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
					log.Println("servers.ServersSetup: Serving API", path)
					hMap[1][path] = singleRegisterHTTPhandler(path, ref)
				}
			}
			ref := strings.Trim(dtn, "_")
			if _, ok := dataMap[ref]; ok {
				log.Println("servers.ServersSetup: Serving API", subpath)
				hMap[1][subpath] = spaceRegisterHTTPhandler(subpath, keysType, ref)
			}
			keysSpaces[spn] = keysType
		}
		p := "/" + strings.Trim(dtn, "_")
		log.Println("servers.ServersSetup: Serving API", p)
		hMap[1][p] = datatypeRegisterHTTPhandler(p, keysSpaces)
	}
	if os.Getenv("MACSTRICT") != "0" {
		strictFlag = true
	} else {
		strictFlag = false
	}
	ports := strings.Split(os.Getenv("HTTPSPORTS"), ",")
	for i, v := range ports {
		if port := strings.Trim(v, " "); port != "" {
			addServer[i] = "0.0.0.0:" + strings.Trim(v, " ")
		} else {
			log.Fatal("servers.ServersSetup: fatal error: invalid addresses")
		}
		for j, c := range addServer {
			if addServer[i] == c && i != j {
				log.Fatal("servers.ServersSetup: fatal error: invalid addresses")
			}
		}
	}
	return nil
}

// StartServers starts all required HTTP/TCP servers
func StartServers() {

	c1 := make(chan bool)      // error quit signal
	c2 := make(chan os.Signal) // quit signal
	ready := false             // it is needed to avoid hanging on c1 before reaching the termination fork

	defer func() {
		if e := recover(); e != nil {
			go func() {
				support.DLog <- support.DevData{"servers.StartServers: recovering server",
					support.Timestamp(), "", []int{1}, true}
			}()
			log.Println("servers.StartServers: recovering from", e)
			// terminating all running servers
			for _, v := range sdServer {
				if v != nil {
					v <- context.Background()
				}
			}
			// terminating the current StartServers
			if ready {
				c1 <- true
			}
			StartServers()
		}
	}()

	if e := setupHTTP(); e != nil {
		log.Println("servers.StartServers: server set-up error:", e)
	} else {

		// Starts first the TCP server for data collection

		ctcp := make(chan context.Context)
		go StartTCP(ctcp)

		// Starts all HTTP service servers

		for i := range addServer {
			// Start HTTP servers
			sdServer[i] = make(chan context.Context)
			startHTTP(addServer[i], sdServer[i], hMap[i])
		}

		sdServer[len(sdServer)-1] = ctcp

		// Two way termination to handle:
		// -  Graceful shutdown when quit via SIGINT (Ctrl+C)
		//    SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
		// - error termination and restart

		signal.Notify(c2, os.Interrupt)
		ready = true
		select {
		case <-c1: // error reported elsewhere, need terminating
		case <-c2: // user termination
			<-c2
			log.Println("servers.StartServers: shutting down")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// Signal shutdown to active servers
			for _, v := range sdServer {
				v <- ctx
			}
			os.Exit(0)
		}
	}
}

// set-up of the TCP server
func setUpTCP() {
	if os.Getenv("CRC") == "1" {
		crcUsed = true
	} else {
		crcUsed = false
	}

	if v, e := strconv.Atoi(os.Getenv("DEVICETO")); e != nil {
		timeout = 5
	} else {
		timeout = v
	}

	if v, e := strconv.Atoi(os.Getenv("MALTO")); e != nil {
		maltimeout = 600
	} else {
		maltimeout = v
	}

	resetbg.start, resetbg.end, resetbg.interval, resetbg.valid = time.Time{}, time.Time{}, time.Duration(0), false
	rng := strings.Split(strings.Trim(os.Getenv("RESETSLOT"), ";"), ";")
	if len(rng) == 3 {
		if v, e := time.Parse(support.TimeLayout, strings.Trim(rng[0], " ")); e == nil {
			resetbg.start = v
			if v, e = time.Parse(support.TimeLayout, strings.Trim(rng[1], " ")); e == nil {
				resetbg.end = v
				if v, e := strconv.Atoi(strings.Trim(rng[2], " ")); e == nil {
					if v != 0 {
						resetbg.interval = time.Duration(v)
						resetbg.valid = true
					}
				}
			}
		}
	}

	log.Println("servers.StartTCP: CRC usage is set to", crcUsed)

	mutexUnknownMac.Lock()
	mutexSensorMacs.Lock()
	sensorChanID = make(map[int]chan []byte)
	SensorCmdMac = make(map[string]chan []byte)
	sensorChanUsedID = make(map[int]bool)
	SensorCmdID = make(map[int]chan []byte)
	sensorMacID = make(map[int][]byte)
	sensorIdMAC = make(map[string]int)
	unknownMacChan = make(map[string]chan net.Conn)
	pendingDevice = make(map[string]bool)
	unkownDevice = make(map[string]bool)
	unusedDevice = make(map[int]string)
	mutexSensorMacs.Unlock()
	mutexUnknownMac.Unlock()

	tcpTokens = make(chan bool, maxsensors)
	for i := 0; i < maxsensors; i++ {
		tcpTokens <- true
	}
}
