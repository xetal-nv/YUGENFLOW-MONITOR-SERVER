package main

import (
	"flag"
	"gateserver/gates"
	"gateserver/sensormodels"
	"gateserver/servers"
	"gateserver/spaces"
	"gateserver/storage"
	"gateserver/support"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const version = "v. 0.9.3b" // version

func main() {
	folder, _ := support.GetCurrentExecDir()
	var cdelay = flag.Int("cdelay", 90, "recovery delay in secs")
	var dbs = flag.String("dbs", "", "databases root folder")
	var dbug = flag.Int("debug", 0, "activate and select a debug mode")
	var dl = flag.Bool("dellogs", false, "delete all logs")
	var de = flag.Bool("dumpentry", false, "dump all entry data to log files")
	var dvl = flag.Bool("dvl", false, "activate dvl")
	var dmode = flag.Int("dmode", 0, "activate and select a development mode")
	var env = flag.String("env", "", "configuration filename")
	var ks = flag.Bool("ks", false, "enable kill switch")
	//var noml = flag.Bool("nomal", false, "disable malicious attack checks")
	var norst = flag.Bool("norst", false, "disable start-up device reset - deprecated now default")
	var forcerst = flag.Bool("forcerst", false, "enable start-up device reset")
	var repcon = flag.Bool("repcon", false, "enable reporting on current data ")
	var ri = flag.Int("ri", 360, "set log rotation interval")
	var rs = flag.Int64("rs", 10000000, "set log rotation size")
	var st = flag.String("start", "", "cstart time expressed as HH:MM")
	var eeprom = flag.Bool("eeprom", false, "enable sensor eeprom refresh at every connection")
	var nosample = flag.Bool("nosample", false, "disable automatic check for database recovery")
	var dbsupdate = flag.Bool("dbsupdate", false, "enable DBS integrity check HTTP API")
	var tcpdeadline = flag.Int("tdl", 24, "TCP read deadline in hours (default 24)")

	flag.Parse()

	log.Printf("Xetal Gate Server version: %v\n", version)

	if *st != "" {
		if ns, err := time.Parse(support.TimeLayout, *st); err != nil {
			log.Println("Syntax error in specified start time", *st)
			os.Exit(1)
		} else {
			now := time.Now()
			nows := strconv.Itoa(now.Hour()) + ":"
			mins := "00" + strconv.Itoa(now.Minute())
			nows += mins[len(mins)-2:]
			if ne, err := time.Parse(support.TimeLayout, nows); err != nil {
				log.Println("Syntax error retrieving system current time")
				os.Exit(1)
			} else {
				del := ns.Sub(ne)
				if del > 0 {
					log.Println("Waiting till", *st, "before starting server")
					time.Sleep(del)
				} else {
					log.Println("!!! WARNING CANNOT WAIT IN THE PAST !!!")
				}
			}
		}
	}

	if *dmode != 0 {
		log.Printf("!!! WARNING DEVELOPMENT MODE %v !!!\n", *dmode)

	}
	if *dbug != 0 && *dbug < 3 {
		log.Printf("!!! WARNING DEBUG MODE %v !!!\n", *dbug)

	} else {
		*dbug = 0
	}
	if *de {
		log.Printf("!!! WARNING DUMP ENTRY IS ENABLED !!!\n")
	}
	if *dl {
		log.Printf("!!! WARNING DELETING ALL LOGS !!!\n")
	}
	if *ks {
		log.Printf("!!! WARNING KILL SWITCH ENABLED !!!\n")
	}
	//if *noml {
	//	log.Printf("!!! WARNING MALICIOUS CHECKS DISABLED !!!\n")
	//}
	if *forcerst {
		log.Printf("!!! WARNING START-UP RESET ENABLED !!!\n")
	}
	if *norst {
		log.Printf("!!! WARNING NORST COMMAND IS DEPRECATED !!!\n")
	}
	if *cdelay != 300 {
		log.Printf("!!! WARNING RECOVERY DELAY CHANGED !!!\n")
	}
	if *repcon {
		log.Printf("!!! WARNING CURRENT REPORTING ENABLED !!!\n")
	}
	if *eeprom {
		log.Printf("!!! WARNING SENSOR EEPROM REFRESH ENABLED !!!\n")
	}
	if !support.SkipDBS {
		if *dbsupdate {
			log.Printf("!!! WARNING DBS INTEGRITY API ENABLED !!!\n")
		}
	} else {
		*dbsupdate = false
	}

	servers.Dvl = *dvl
	support.Debug = *dbug
	support.RotInt = *ri
	support.RotSize = *rs
	support.DelLogs = *dl
	support.MalOn = false
	support.RstON = *forcerst
	spaces.CrashMaxDelay = int64(*cdelay) * 1000
	servers.KSwitch = *ks
	servers.RepCon = *repcon
	gates.LogToFileAll = *de
	servers.SensorEEPROMResetEnabled = *eeprom
	servers.EnableDBSApi = *dbsupdate
	servers.TCPdeadline = *tcpdeadline

	folder = os.Getenv("GATESERVER")

	var e error
	if folder != "" {
		e = os.Chdir(folder)
	}

	cleanup := func() {
		// we use the crash timestamp as timestamp for all data
		log.Println("Saving latest values for recovery")
		if f, err := os.Create(".recoveryavg"); err != nil {
			log.Printf("RECOVERY DATA SAMPLER ERROR %v \n", err.Error())
		} else {
			//noinspection GoUnhandledErrorResult
			defer f.Close()
			cTS := support.Timestamp()
			// Saves the latest sampler values
			for sam, el0 := range spaces.LatestBankOut {
				for sp, el1 := range el0 {
					for ms, ch := range el1 {
						var ok bool
						data := sam + "," + sp + "," + ms + ","
						switch strings.Trim(sam, "_") {
						case "entry":
							dt := new(storage.SeriesEntries)
							_ = dt.ExtractForRecovery(<-ch)
							//ok = dt.Tag() != "" && dt.Ts() != 0
							if ok = dt.Tag() != "" && dt.Ts() != 0; ok {
								data += strconv.FormatInt(cTS, 10) + ","
								if ok = len(dt.Sval) != 0; ok {
									data += "["
									val := dt.Sval
									//fmt.Println(val)
									for i := 0; i < len(val); i++ {
										data += "[" + strconv.Itoa(val[i][0]) + " " + strconv.Itoa(val[i][1]) +
											" " + strconv.Itoa(val[i][2]) + " " + strconv.Itoa(val[i][3]) + "]"
									}
									data += "]\n"
								}
							}
						case "sample":
							dt := new(storage.SeriesSample)
							_ = dt.Extract(<-ch)
							if ok = dt.Tag() != "" && dt.Ts() != 0; ok {
								data += strconv.FormatInt(cTS, 10) + ","
								data += strconv.Itoa(dt.Val()) + "\n"
							}
						default:
						}
						if ok {
							if _, err := f.WriteString(data); err != nil {
								log.Printf("RECOVERY DATA SAMPLER ERROR %v \n", err.Error())
							}
						}
					}
				}
			}
		}
		if f, err := os.Create(".recoverypres"); err != nil {
			log.Printf("RECOVERY DATA DETECTORS ERROR %v \n", err.Error())
		} else {
			//noinspection GoUnhandledErrorResult
			defer f.Close()
			// Saves the latest detector values
			for space, detVal := range spaces.LatestDetectorOut {
				if detVal != nil {
					val := <-detVal
					if val != nil {
						for _, el := range val {
							data := space + "," + el.Id + "," + strconv.FormatInt(el.Start.Unix(), 10) + "," +
								strconv.FormatInt(el.End.Unix(), 10) + "," + strconv.FormatInt(time.Now().Unix(), 10) +
								"," + strconv.Itoa(el.Activity.NetFlow) + "\n"
							if _, err := f.WriteString(data); err != nil {
								log.Printf("RECOVERY DATA DETECTORS ERROR %v \n", err.Error())
							}
							//fmt.Println(space, el.Id, el.Start.Unix(), el.End.Unix(), el.Activity.Ts, el.Activity.NetFlow)
						}
					}
				}
			}
		}
		log.Println("System shutting down")
		support.Terminate()
		storage.TimedIntDBSClose()
	}
	support.SetUp(*env)

	if folder != "" {
		if e == nil {
			log.Printf("Move to folder %v\n", folder)
		} else {
			log.Fatalf("Unable to move to folder %v, error reported:%v\n", folder, e)
		}
	}

	// Set-up databases
	if err := storage.TimedIntDBSSetUp(*dbs, false); err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	gates.SetUp()
	spaces.SetUp()

	if !support.SkipDBS {
		storage.RetrieveSampleFromFile(!*nosample)
		storage.RetrievePresenceFromFile(!*nosample)
	}

	switch *dmode {
	case 3:
		go func() {
			time.Sleep(10 * time.Second)
			sensormodels.Office()
		}()
	case 2:
		for i := 100; i < 300; i++ {
			mac := []byte{'a', 'b', 'c'}
			mac = append(mac, []byte(strconv.Itoa(i))...)
			go func(i int, mac []byte) {
				time.Sleep(time.Duration(rand.Intn(360)) * time.Second)
				sensormodels.SensorModel(i-100, 5000000, 100, []int{-1, 0, 1}, mac)
			}(i, mac)
		}
		for i := 300; i < 450; i++ {
			mac := []byte{'a', 'b', 'c'}
			mac = append(mac, []byte(strconv.Itoa(i))...)
			go func(i int, mac []byte) {
				time.Sleep(time.Duration(rand.Intn(360)) * time.Second)
				sensormodels.SensorModel(65535, 500, 60, []int{-1, 0, 1}, mac)
			}(i, mac)
		}
	case 1:
		go sensormodels.SensorModel(0, 7000, 3, []int{-1, 1}, []byte{'a', 'b', 'c', '1', '2', '1'})
		go sensormodels.SensorModel(1, 7000, 30, []int{-1, 1}, []byte{'a', 'b', 'c', '1', '2', '2'})
		//go sensormodels.SensorModel(20, 5000, 20, []int{-1, 1}, []byte{'a', 'b', 'c', '1', '2', '3'})
		//go sensormodels.SensorModel(21, 5500, 30, []int{-1, 1}, []byte{'a', 'b', 'c', '1', '2', '4'})
		//go sensormodels.SensorModel(2340, 900, 20, []int{-1, 1}, []byte{'a', 'b', 'c', '1', '2', '5'})
		//go sensormodels.SensorModel(65535, 500, 20, []int{-1, 1}, []byte{'a', 'b', 'c', '1', '2', '6'})
	default:
	}

	// Capture all killing s
	c := make(chan os.Signal)
	//signal.Notify(c)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL,
		syscall.SIGQUIT, syscall.SIGABRT)
	go func() {
		<-c
		support.CleanupLock.Lock()
		cleanup()
		support.CleanupLock.Unlock()
		os.Exit(1)
	}()

	// Set-up and start servers
	servers.StartServers()

}
