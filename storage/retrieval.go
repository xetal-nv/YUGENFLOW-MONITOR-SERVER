package storage

import (
	"bufio"
	"fmt"
	"gateserver/support"
	"os"
	"strconv"
	"strings"
	"time"
)

const MAXAGE = 2
const FORM = "January 2 2006"

// covert a string to its epoch value using as form FORM
func string2epoch(val string) (rt int64, err error) {
	// Parse the string according to the form.
	r, e := time.Parse(FORM, val)
	if e == nil {
		rt = r.Unix()
	}
	err = e
	return
}

// start database sample values retrieval following format given in .recoverysamples
// this includes also removal of all sample data in the given interval
// the recovery files needs to respect the following format. For each day to be retrieved
// day, space_name, measurement_name, [time24h/value]
// Example
// January 15 2019, living, 20min, 10:30/15, 14:30/10
// Furthermore, recovery files will be rejected of older than MAXAGE hours
// Comments in the recovery file need to start with the symbol #
func RetrieveSampleFromFile() {
	//fmt.Println("retrieveSampleFromFile")

	if file, err := os.Open(".recoverysamples"); err != nil {
		go func() {
			support.DLog <- support.DevData{"storage.RetrieveSampleFromFile", support.Timestamp(), "err", []int{1}, true}
		}()
	} else {
		defer file.Close()
		//var newData [][]string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if string(strings.Trim(scanner.Text(), " ")[0]) != "#" {
				lineData := strings.Split(scanner.Text(), ",")
				if v, e := string2epoch(lineData[0]); e == nil {
					//lineData[0] = strconv.Itoa(int(v))
					//newData = append(newData, lineData)
					label := support.StringLimit("sample", support.LabelLength)
					label += support.StringLimit(strings.Trim(lineData[1], " "), support.LabelLength)
					label += support.StringLimit(strings.Trim(lineData[2], " "), support.LabelLength)
					s0 := &SerieSample{Stag: label, Sts: v * 1000}
					s1 := &SerieSample{Stag: label, Sts: (v + 86399) * 1000}
					//fmt.Println(s0,s1)
					//var rt []SampleData
					//fmt.Println("New cycle")
					//if tag, ts, vals, e := ReadSeriesTS(s0, s1, true); e == nil {
					//	//rt = s0.UnmarshalSliceSS(tag, ts, vals)
					//	//fmt.Println(rt)
					//	fmt.Println(tag, ts, vals)
					//}
					//fmt.Println(DeleteSeriesTS(s0, s1, true))
					if err := DeleteSeriesTS(s0, s1, true); err == nil {
						var testData []SerieSample
						for i := 3; i < len(lineData); i++ {
							sampleRaw := strings.Split(strings.Trim(lineData[i], " "), "/")
							//fmt.Println(sampleRaw)
							if th, err := strconv.ParseInt(strings.Split(sampleRaw[0], ":")[0], 10, 64); err == nil {
								th *= 3600
								if tm, err := strconv.ParseInt(strings.Split(sampleRaw[0], ":")[1], 10, 64); err == nil {
									tm *= 60
									if val, err := strconv.Atoi(sampleRaw[1]); err == nil {
										newSample := new(SerieSample)
										newSample.Stag = label
										newSample.Sts = (th + tm + v) * 1000
										newSample.Sval = val
										testData = append(testData, *newSample)
										//fmt.Println(newSample)
										//fmt.Println(StoreSampleTS(newSample, true))
										if err := StoreSampleTS(newSample, true); err != nil {
											go func() {
												support.DLog <- support.DevData{"storage.RetrieveSampleFromFile", support.Timestamp(), "error writing data",
													[]int{1}, true}
											}()
										}
									}
								}
							}

						}
						if tag, ts, vals, e := ReadSeriesTS(s0, s1, true); e == nil {
							readData := s0.UnmarshalSliceSS(tag, ts, vals)
							if len(readData) != len(testData) {
								go func() {
									support.DLog <- support.DevData{"storage.RetrieveSampleFromFile", support.Timestamp(), "error writing all data",
										[]int{1}, true}
								}()
							} else {
								go func() {
									support.DLog <- support.DevData{"storage.RetrieveSampleFromFile", support.Timestamp(), "retieved sample data",
										[]int{1}, true}
								}()
							}
						}
					} else {
						go func() {
							support.DLog <- support.DevData{"storage.RetrieveSampleFromFile", support.Timestamp(), "error deleting data", []int{1}, true}
						}()
					}
				}
			}
			// newData is an array containing the DBS data with epoch instead of the readable data format as in FORM
			//fmt.Println(newData)

			if err := scanner.Err(); err != nil {
				go func() {
					support.DLog <- support.DevData{"storage.RetrieveSampleFromFile", support.Timestamp(), "error reading .recoverysamples", []int{1}, true}
				}()
			}
		}
	}
}

// start database sample values retrieval following format given in .recoverypresence
// this includes also removal of all sample data in the given interval
func RetrievePresenceFromFile() {
	// TODO everything
	fmt.Println("RetrievePresenceFromFile")
}
