package servers

import (
	"bufio"
	"context"
	"errors"
	"gateserver/spaces"
	"gateserver/storageold"
	"gateserver/supp"
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

var Dvl = false

func setJSenv() {
	if supp.DisableWebApp {
		return
	}
	if dat, e := ioutil.ReadFile("dbs/dat"); e == nil {
		f, err := os.Create("./html/js/dat.js")
		if err != nil {
			log.Fatal("Fatal error creating dat.js: ", err)
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
		log.Fatal("servers.setJSenv: fatal error cannot retrieve dbs/dat")
	}

	//ports := strings.Split(os.Getenv("HTTPSPORTS"), ",")
	//ports := os.Getenv("HTTPSPORTS")
	addServer = "0.0.0.0:" + strings.Trim(os.Getenv("HTTPSPORTS"), " ")
	//for i, v := range ports {
	//	if port := strings.Trim(v, " "); port != "" {
	//		addServer[i] = "0.0.0.0:" + strings.Trim(v, " ")
	//	} else {
	//		log.Fatal("ServersSetup: fatal error: invalid addresses")
	//	}
	//if !mergeHTMLservers {
	//	for j, c := range addServer {
	//		if addServer[i] == c && i != j {
	//			log.Fatal("ServersSetup: fatal error: invalid addresses")
	//		}
	//	}
	//}
	//}

	f, err := os.Create("./html/js/def.js")
	if err != nil {
		log.Fatal("Fatal error creating def.js: ", err)
	}
	//}

	//ip := ""
	//if ip = os.Getenv("IP"); ip == "" {
	//	ip = supp.GetOutboundIP().String()
	//}

	//js := "var ip = \"http://" + ip + ":" + strings.Trim(ports[len(ports)-1], " ") + "\";\n"
	//js := "var port = \"" + strings.Trim(ports[len(ports)-1], " ") + "\";\n"
	//if _, err := f.WriteString(js); err != nil {
	//	_ = f.Close()
	//	log.Fatal("Fatal error writing to def.js: ", err)
	//}

	js := "var samplingWindow = " + strconv.Itoa(spaces.SamplingWindow) + " * 1000;\n"
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	if RepCon {
		js = "var reportCurrent = true;\n"
	} else {
		js = "var reportCurrent = false;\n"
	}
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	js = "var edition = \"" + strings.Trim(os.Getenv("EDITION"), " ") + "\";\n"
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	js = "var spaceTimes = {\n"
	for i, v := range spaces.SpaceTimes {
		name := strings.Trim(i, "_")
		hourS := "0" + strconv.Itoa(v.Start.Hour())
		minS := "0" + strconv.Itoa(v.Start.Minute())
		hourE := "0" + strconv.Itoa(v.End.Hour())
		minE := "0" + strconv.Itoa(v.End.Minute())
		js += "\"" + name + "\": " + "[\"" + hourS + ":" + minS + "\", \"" + hourE + ":" + minE + "\"],\n"
	}
	js += "};\n"
	js += "var labellength = " + strconv.Itoa(supp.LabelLength) + ";\n"
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	js = "var overviewReport = false;\n"
	if strings.Trim(os.Getenv("OVERVIEWREPORT"), " ") != "" &&
		strings.Trim(os.Getenv("OVERVIEWDATA"), " ") != "" {
		jsAlt := "var overviewReport = true;\nlet overviewReportDefs = ["
		for _, el := range strings.Split(strings.Trim(os.Getenv("OVERVIEWDATA"), " "), ";") {
			eldef := strings.Split(strings.Trim(el, " "), " ")
			//fmt.Println(eldef)
			switch eldef[0] {
			case "point":
				jsAlt += "{name: \"at " + eldef[1] + "\", start: \"\", end: \"\", point: \"" + eldef[1] +
					"\", precision: \"" + eldef[2] + "\", presence: \"\", id: 0},\n"
			case "period":
				jsAlt += "{name: \"" + eldef[1] + " to " + eldef[2] + "\", start: \"" + eldef[1] +
					"\", end: \"" + eldef[2] + "\", point: \"\", precision: \"\", presence: \"\", id: 0},\n"
			case "presence":
				jsAlt += "{name: \"activity " + eldef[1] + " to " + eldef[2] + "?\", start: \"" + eldef[1] +
					"\", end: \"" + eldef[2] + "\", point: \"" +
					"\", precision: \"\", presence: \"" + eldef[3] + "\", id: 0},\n"
			case "":
			default:
				log.Fatal("Fatal error in OVERVIEWDATA, illegal value in  ", el)
			}
		}
		if schd := strings.Split(strings.Trim(os.Getenv("OVERVIEWREPORT"), " "), " "); len(schd) == 2 {
			jsAlt += "{name: \"day\", start: \"" + schd[0] + "\", end: \"" + schd[1] + "\", point: \"\", precision: \"\", " +
				"presence: \"\", id: 0, skip: true}];\n"
		} else {
			log.Fatal("Fatal error in OVERVIEWREPORT, illegal value")
		}
		if val := strings.Trim(os.Getenv("REFERENCESAMPLES"), " "); val != "" {
			jsAlt += "var refOverviewAsys = \"" + val + "\";\n"
			js = jsAlt
		}
		if val := strings.Trim(os.Getenv("SKIPDAYS"), " "); val != "" {
			jsAlt += "var overviewSkipDays = ["
			for _, v := range strings.Split(val, " ") {
				jsAlt += "\"" + v + "\", "
			}
			js = jsAlt[:len(jsAlt)-2] + "];\n"
		} else {
			js += "var overviewSkipDays = [];\n"
		}
	}
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	if val := strings.Trim(os.Getenv("RTSHOW"), " "); val != "" {
		ll := strings.Split(val, " ")
		js = "var rtshow = ["
		for _, v := range ll {
			js += "\"" + strings.Trim(v, " ") + "\", "
		}
		js = js[0 : len(js)-2]
		js += "];\n"
	} else {
		js = "var rtshow = [];\n"
	}
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	if val := strings.Trim(os.Getenv("REPSHOW"), " "); val != "" {
		ll := strings.Split(val, " ")
		js = "var repshow = ["
		for _, v := range ll {
			js += "\"" + strings.Trim(v, " ") + "\", "
		}
		js = js[0 : len(js)-2]
		js += "];\n"

	} else {
		js = "var repshow = \"\";\n"
	}
	if _, err := f.WriteString(js); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	jsTxt := "var openingTime = \"\";\n"
	jsST := "var opStartTime = \"\";\n"
	jsEN := "var opEndTime = \"\";\n"

	if strings.Trim(os.Getenv("RTWINDOW"), " ") == "" {
		if val := strings.Split(strings.Trim(os.Getenv("ANALYSISWINDOW"), " "), " "); len(val) == 2 {
			if _, e := time.Parse(supp.TimeLayout, val[0]); e == nil {
				if _, e := time.Parse(supp.TimeLayout, val[1]); e == nil {
					jsTxt = "var openingTime = \"from " + val[0] + " to " + val[1] + "\";\n"
					jsST = "var opStartTime = \"" + val[0] + "\";\n"
					jsEN = "var opEndTime = \"" + val[1] + "\";\n"
					log.Printf("spaces.setJSenv: Analysis window is set from %v to %v\n", val[0], val[1])
				} else {
					log.Fatal("spaces.setJSenv: illegal end ANALYSISWINDOW value", val)
				}
			} else {
				log.Fatal("spaces.setJSenv: illegal start ANALYSISWINDOW value", val)
			}
		}
	} else {
		if val := strings.Split(strings.Trim(os.Getenv("RTWINDOW"), " "), " "); len(val) == 2 {
			if _, e := time.Parse(supp.TimeLayout, val[0]); e == nil {
				if _, e := time.Parse(supp.TimeLayout, val[1]); e == nil {
					jsTxt = "var openingTime = \"from " + val[0] + " to " + val[1] + "\";\n"
					jsST = "var opStartTime = \"" + val[0] + "\";\n"
					jsEN = "var opEndTime = \"" + val[1] + "\";\n"
					log.Printf("spaces.setJSenv: Analysis window is set from %v to %v\n", val[0], val[1])
				} else {
					log.Fatal("spaces.setJSenv: illegal end RTWINDOW value", val)
				}
			} else {
				log.Fatal("spaces.setJSenv: illegal start RTWINDOW value", val)
			}
		}
	}

	if _, err := f.WriteString(jsTxt); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}
	if _, err := f.WriteString(jsST); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}
	if _, err := f.WriteString(jsEN); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	jsTxt = "var spaceDefinitions = {"
	for nm, df := range spaces.SpaceDef {
		nm = strings.Replace(nm, "_", "", -1)
		jsTxt += "\"" + nm + "\": ["
		for _, en := range df {
			jsTxt += strconv.Itoa(en) + ","
		}
		jsTxt = strings.Trim(jsTxt, ",") + "],"
	}
	jsTxt = strings.Trim(jsTxt, ",") + "};"
	if _, err := f.WriteString(jsTxt); err != nil {
		_ = f.Close()
		log.Fatal("Fatal error writing to def.js: ", err)
	}

	if err = f.Close(); err != nil {
		log.Fatal("Fatal error closing def.js: ", err)
	}
}

// set-up of HTTP servers and handlers
func setupHTTP() error {

	setJSenv()

	dataMap = make(map[string]dataFunc)
	dataMap["sample"] = func() GenericData { return new(storageold.SeriesSample) }
	dataMap["entry"] = func() GenericData { return new(storageold.JsonSeriesEntries) }

	dbgMutex.Lock()
	dbgRegistry = make(map[string]int64)
	dbgMutex.Unlock()

	// enable web server - if ports are different
	//fmt.Println(addServer)
	//if addServer[0] == addServer[1] {
	//	mergeHTMLservers = true
	//} else {
	//	hMap[0] = map[string]http.Handler{
	//		"./html/": nil,
	//	}
	//}
	//os.Exit(1)

	hMap = make(map[string]http.Handler)
	// development log API
	if Dvl {
		hMap["/dvl"] = dvlHTTHandler()
		log.Println("WARNING: Developer Logfile enabled")
	}
	// installation information API
	hMap["/info"] = infoHTTHandler()
	if !supp.SkipDBS {
		// Series data retrieval API
		hMap["/series"] = seriesHTTPhandler()
		// Presence data retrieval API
		hMap["/presence"] = presenceHTTPhandler()
	}
	// Sensor command API
	hMap["/command"] = commandHTTHandler()
	// analysis information API
	hMap["/asys"] = asysHTTHandler()
	// unused registered device API
	hMap["/und"] = unusedDeviceHTTPHandler()
	// pending registered device API
	hMap["/pending"] = pendingDeviceHTTPHandler()
	// unknown registered device API and its variants
	hMap["/udef"] = undefinedDeviceHTTPHandler("")
	hMap["/udef/active"] = undefinedDeviceHTTPHandler("active")
	hMap["/udef/defined"] = undefinedDeviceHTTPHandler("defined")
	hMap["/udef/undefined"] = undefinedDeviceHTTPHandler("undefined")
	hMap["/udef/notactive"] = undefinedDeviceHTTPHandler("notactive")
	// unused registered device API
	hMap["/active"] = usedDeviceHTTPHandler()
	// if enabled it register the kill switch API
	if KSwitch {
		hMap["/ks"] = killswitchHTTPHandler()
	}
	// Api for dbs management
	if EnableDBSApi {
		hMap["/dbs/retrieve/samples"] = retrieveDBSsamples()
		hMap["/dbs/retrieve/presence"] = retrieveDBSpresence()
	}

	if !supp.DisableWebApp {
		// add SVG API for installation graphs
		for spn := range spaces.SpaceDef {
			name := strings.Replace(spn, "_", "", -1)
			hMap["/plan/"+name] = planHTTPHandler(name)
		}
		hMap["/plan/logo"] = planHTTPHandler("logo")
	}

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
					hMap[path] = singleRegisterHTTPhandler(path, ref)
				}
			}
			ref := strings.Trim(dtn, "_")
			if _, ok := dataMap[ref]; ok {
				log.Println("servers.ServersSetup: Serving API", subpath)
				hMap[subpath] = spaceRegisterHTTPhandler(subpath, keysType, ref)
			}
			keysSpaces[spn] = keysType
		}
		p := "/" + strings.Trim(dtn, "_")
		log.Println("servers.ServersSetup: Serving API", p)
		hMap[p] = datatypeRegisterHTTPhandler(p, keysSpaces)
	}
	if os.Getenv("MACSTRICT") != "0" {
		strictFlag = true
	} else {
		strictFlag = false
	}
	addServer = "0.0.0.0:" + strings.Trim(os.Getenv("HTTPSPORTS"), " ")
	//ports := strings.Split(os.Getenv("HTTPSPORTS"), ",")
	//for i, v := range ports {
	//	if port := strings.Trim(v, " "); port != "" {
	//		addServer[i] = "0.0.0.0:" + strings.Trim(v, " ")
	//	} else {
	//		log.Fatal("servers.ServersSetup: fatal error: invalid addresses")
	//	}
	//	//for j, c := range addServer {
	//	//	if addServer[i] == c && i != j {
	//	//		log.Fatal("servers.ServersSetup: fatal error: invalid addresses")
	//	//	}
	//	//}
	//}
	return nil
}

/*
	setSensorParameters read all sensor parameters ignoring any data being sent.
	The data is contained in the file .sensorsettings and each line specifies on sensor as:

	{mac} {srate} {savg} {bgth*16} {occth*16}

	- comments start with #
	- a mac set to * (wildcard) indicated settings valid for every sensor
	- a setting set to _ either takes the value from the mac wildcard (if given) or ignore the setting

*/
func readSensorParameters() {

	if SensorEEPROMResetEnabled {
		if supp.FileExists(sensorEEPROMfile) {
			if v, e := strconv.Atoi(os.Getenv("EEPROMDELAY")); e != nil {
				sensorEEPROMResetDelay = 10
			} else {
				sensorEEPROMResetDelay = v
			}
			if v, e := strconv.Atoi(os.Getenv("EEPROMSTEP")); e != nil {
				sensorEEPROMResetStep = 10
			} else {
				sensorEEPROMResetStep = v
			}

			log.Printf("EEPROM refresh rate is set at %v with delay %v", sensorEEPROMResetStep, sensorEEPROMResetDelay)
			log.Println("Setting sensor settings from file " + sensorEEPROMfile)

			refGen := false
			commonSensorSpecsLocal := sensorSpecs{0, 0, 0, 0}
			sensorDataLocal := make(map[string]sensorSpecs)

			readSpecs := func(line string, ref bool) (specs sensorSpecs, el string, refGen bool, e error) {
				refGen = ref
				values := strings.Split(line, " ")
				if len(values) != 5 {
					log.Println("Error illegal sensor settings:", line)
				} else {
					el = strings.Trim(values[0], " ")
					var e1, e2, e3, e4 error
					if strings.Trim(values[1], " ") == "_" {
						refGen = true
						specs.srate = -1
					} else {
						specs.srate, e1 = strconv.Atoi(values[1])
					}
					if strings.Trim(values[2], " ") == "_" {
						refGen = true
						specs.savg = -1
					} else {
						specs.savg, e2 = strconv.Atoi(values[2])
					}
					if strings.Trim(values[3], " ") == "_" {
						refGen = true
						specs.bgth = -1
					} else {
						specs.bgth, e3 = strconv.ParseFloat(values[3], 64)
					}
					if strings.Trim(values[4], " ") == "_" {
						refGen = true
						specs.occth = -1
					} else {
						specs.occth, e4 = strconv.ParseFloat(values[4], 64)
					}

					if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
						//log.Println("Error illegal sensor settings:", line)
						return sensorSpecs{}, "", ref, errors.New("Error illegal sensor settings: " + line)
					}

					if specs.srate == 0 || specs.savg == 0 || specs.bgth == 0 || specs.occth == 0 {
						//log.Println("Error illegal sensor settings:", line)
						return sensorSpecs{}, "", ref, errors.New("Error illegal sensor setting values: " + line)
					}

					if el == "*" && (specs.srate == -1 || specs.savg == -1 || specs.bgth == -1 || specs.occth == -1) {
						//log.Println("Error illegal sensor settings:", line)
						return sensorSpecs{}, "", ref, errors.New("Error illegal sensor setting values: " + line)
					}

				}
				return
			}

			expandSpecs := func() error {
				if commonSensorSpecsLocal.srate == 0 || commonSensorSpecsLocal.savg == 0 || commonSensorSpecsLocal.bgth == 0 || commonSensorSpecsLocal.occth == 0 {
					//log.Println("Error illegal sensor settings:", line)
					return errors.New("error illegal global sensor setting values")
				}

				for i, val := range sensorDataLocal {
					if val.srate == -1 {
						val.srate = commonSensorSpecsLocal.srate
					}
					if val.savg == -1 {
						val.savg = commonSensorSpecsLocal.savg
					}
					if val.bgth == -1 {
						val.bgth = commonSensorSpecsLocal.bgth
					}
					if val.occth == -1 {
						val.occth = commonSensorSpecsLocal.occth
					}
					sensorDataLocal[i] = val
				}

				return nil
			}

			if file, err := os.Open(sensorEEPROMfile); err == nil {

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.Trim(scanner.Text(), "")
					if string(line[0]) != "#" {
						if tempSpecs, id, refGenTmp, err := readSpecs(line, refGen); err != nil {
							log.Println(err.Error())
						} else {
							refGen = refGenTmp
							if id == "*" {
								commonSensorSpecsLocal = tempSpecs
							} else {
								sensorDataLocal[id] = tempSpecs
							}
						}
					}
				}
				if err := scanner.Err(); err != nil {
					log.Fatal("Error reading setting sensor settings from file " + sensorEEPROMfile)
				}

				commonSensorSpecs = commonSensorSpecsLocal
				if refGen {
					if err := expandSpecs(); err == nil {
						//commonSensorSpecs = commonSensorSpecsLocal
						sensorData = sensorDataLocal
					} else {
						log.Fatal(err.Error())
					}
				}
				//noinspection GoUnhandledErrorResult
				file.Close()
			} else {
				log.Fatal("Error opening setting sensor settings from file " + sensorEEPROMfile)
			}
		} else {
			log.Fatal("File " + sensorEEPROMfile + " does not exists")
		}
	}
}

// StartServers starts all required HTTP/TCP servers
func StartServers() {

	c1 := make(chan bool)      // error quit signal
	c2 := make(chan os.Signal) // quit signal
	ready := false             // it is needed to avoid hanging on c1 before reaching the termination fork

	defer func() {
		if e := recover(); e != nil {
			go func() {
				supp.DLog <- supp.DevData{"servers.StartServers: recovering server",
					supp.Timestamp(), "", []int{1}, true}
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
		// read Sensor specs
		readSensorParameters()

		// Starts first the TCP server for data collection

		ctcp := make(chan context.Context)
		go StartTCP(ctcp)

		// Starts all HTTP service servers
		startHTTP(addServer, sdServer[0], hMap)

		//for i := range addServer {
		//	// Start HTTP servers
		//	if len(hMap[i]) == 0 {
		//		log.Printf("servers.StartServers: skipping server %v since it serves no paths\n", addServer[i])
		//	} else {
		//		sdServer[i] = make(chan context.Context)
		//		startHTTP(addServer[i], sdServer[i], hMap[i])
		//	}
		//}

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
		malTimeout = 600
	} else {
		malTimeout = v
	}

	resetBG.start, resetBG.end, resetBG.valid = time.Time{}, time.Time{}, false
	rng := strings.Split(strings.Trim(os.Getenv("RESETSLOT"), ";"), ";")

	if v, e := strconv.Atoi(strings.Trim(os.Getenv("RESETPERIOD"), " ")); e == nil {
		if v != 0 {
			resetBG.interval = time.Duration(v)
			if len(rng) == 3 {
				if v, e := time.Parse(supp.TimeLayout, strings.Trim(rng[0], " ")); e == nil {
					resetBG.start = v
					if v, e = time.Parse(supp.TimeLayout, strings.Trim(rng[1], " ")); e == nil {
						resetBG.end = v
						if strings.Trim(rng[2], " ") != "0" {
							resetBG.valid = true
						}
					}
				}
			}
		}
	}

	if !resetBG.valid {
		log.Println("servers.StartTCP: WARNING RESETSLOT has invalid data or is disabled:", os.Getenv("RESETSLOT"))
	} else {
		log.Println("servers.StartTCP: WARNING RESETSLOT is set from ", strings.Trim(rng[0], " "), "to", strings.Trim(rng[1], " "))
	}

	log.Println("servers.StartTCP: CRC usage is set to", crcUsed)

	mutexUnknownMac.Lock()
	mutexSensorMacs.Lock()
	sensorChanID = make(map[int]chan []byte)
	SensorCmdMac = make(map[string][]chan []byte)
	SensorIDCMDMac = make(map[string]chan int)
	sensorChanUsedID = make(map[int]bool)
	SensorCmdID = make(map[int]chan []byte)
	sensorConnMAC = make(map[string]net.Conn)
	sensorMacID = make(map[int][]byte)
	sensorIdMAC = make(map[string]int)
	unknownMacChan = make(map[string]chan net.Conn)
	pendingDevice = make(map[string]bool)
	unknownDevice = make(map[string]bool)
	unusedDevice = make(map[int]string)
	mutexSensorMacs.Unlock()
	mutexUnknownMac.Unlock()

	tcpTokens = make(chan bool, maxSensors)
	for i := 0; i < maxSensors; i++ {
		tcpTokens <- true
	}
}
