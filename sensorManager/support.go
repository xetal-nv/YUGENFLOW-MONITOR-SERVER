package sensorManager

import (
	"bufio"
	"errors"
	"fmt"
	"gateserver/storage/diskCache"
	"gateserver/support/globals"
	"gateserver/support/others"
	"github.com/fpessolano/mlogger"
	"log"
	"os"
	"strconv"
	"strings"
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
			fmt.Printf("*** INFO: setID DoS check started %v:%v ***\n", globals.SecurityLength, globals.RepetitiveTimeout)
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
		_, _ = diskCache.MarkIP([]byte(ipc), globals.MaliciousTriesIP)
		_, _ = diskCache.MarkMAC([]byte(mac), globals.MaliciousTriesMac)
		return true
	}
}

/*
	setSensorParameters read all sensor parameters ignoring any data being sent.
	The data is contained in the file sensors.settings and each line specifies on sensor as:

	{mac} {srate} {savg} {bgth*16} {occth*16}

	- comments start with #
	- a mac set to * (wildcard) indicated settings valid for every sensor
	- a setting set to _ either takes the value from the mac wildcard (if given) or ignore the setting

*/
func LoadSensorEEPROMSettings() {

	if globals.SensorEEPROMResetEnabled {
		if others.FileExists(globals.SensorSettingsFile) {

			fmt.Printf("*** INFO: EEPROM refresh rate is set at %v with delay %v ***\n", globals.SensorEEPROMResetStep, globals.SensorEEPROMResetDelay)
			fmt.Printf("*** INFO: Setting sensor settings from file %v ***\n", globals.SensorSettingsFile)

			refGen := false
			commonSensorSpecs = sensorSpecs{0, 0, 0, 0}
			sensorData = make(map[string]sensorSpecs)

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
				if commonSensorSpecs.srate == 0 || commonSensorSpecs.savg == 0 || commonSensorSpecs.bgth == 0 || commonSensorSpecs.occth == 0 {
					//log.Println("Error illegal sensor settings:", line)
					return errors.New("error illegal global sensor setting values")
				}

				for i, val := range sensorData {
					if val.srate == -1 {
						val.srate = commonSensorSpecs.srate
					}
					if val.savg == -1 {
						val.savg = commonSensorSpecs.savg
					}
					if val.bgth == -1 {
						val.bgth = commonSensorSpecs.bgth
					}
					if val.occth == -1 {
						val.occth = commonSensorSpecs.occth
					}
					sensorData[i] = val
				}

				return nil
			}

			if file, err := os.Open(globals.SensorSettingsFile); err == nil {

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.Trim(scanner.Text(), "")
					if string(line[0]) != "#" {
						if tempSpecs, id, refGenTmp, err := readSpecs(line, refGen); err != nil {
							log.Println(err.Error())
						} else {
							refGen = refGenTmp
							if id == "*" {
								commonSensorSpecs = tempSpecs
							} else {
								id = strings.Replace(id, ":", "", -1)
								sensorData[id] = tempSpecs
							}
						}
					}
				}
				if err := scanner.Err(); err != nil {
					log.Fatal("Error reading setting sensor settings from file " + globals.SensorSettingsFile)
				}

				fmt.Println("EEPROM Reference definition: ", commonSensorSpecs)
				if refGen {
					if err := expandSpecs(); err == nil {
						if globals.DebugActive {
							fmt.Printf("EEPROM Sensor definitions: %+v\n", sensorData)
						}
					} else {
						log.Fatal(err.Error())
					}
				}
				//noinspection GoUnhandledErrorResult
				file.Close()
			} else {
				log.Fatal("Error opening setting sensor settings from file " + globals.SensorSettingsFile)
			}
		} else {
			log.Fatal("File " + globals.SensorSettingsFile + " does not exists")
		}
	}
}
