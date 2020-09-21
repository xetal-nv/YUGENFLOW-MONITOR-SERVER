package storage

import (
	"bufio"
	"gateserver/supp"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const MAXAGE = 2
const FORM = "January 2 2006"
const SAMPLEFILE = ".recoverysamples"
const PRESENCEFILE = ".recoverypresence"

// covert a string to its epoch value using as form FORM
func string2epoch(val string) (rt int64, err error) {
	// Parse the string according to the form.
	r, e := time.ParseInLocation(FORM, val, time.Now().Location())
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
// Furthermore, recovery files will be rejected of older than MAXAGE hours (TBD)
// Comments in the recovery file need to start with the symbol #
func RetrieveSampleFromFile(startup bool) {
	//fmt.Println("retrieveSampleFromFile")

	sampleCorruptions := 0
	flowCorruptions := 0

	if fileStat, err := os.Stat(SAMPLEFILE); err == nil {
		if time.Now().Unix()-fileStat.ModTime().Unix() > MAXAGE*3600 && !startup {
			go func() {
				supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "attempted to use an old .recoverysamples", []int{1}, true}
			}()
			return
		}
		if startup {
			log.Println("!!! WARNING SAMPLE DATABASE INTEGRITY CHECK INITIATED !!!")
		}
	} else {
		if startup {
			go func() {
				supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "error opening file .recoverysamples", []int{1}, true}
			}()
			return
		} else {
			log.Println("!!! WARNING SAMPLE DATABASE INTEGRITY IS OK !!!")
			return
		}
	}

	if file, err := os.Open(SAMPLEFILE); err != nil {
		go func() {
			supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "error opening file .recoverysamples", []int{1}, true}
		}()
		if startup {
			log.Println("!!! WARNING SAMPLE DATABASE RECOVERY ERROR !!!")
		}
	} else {
		//noinspection GoUnhandledErrorResult
		//defer file.Close()
		//var newData [][]string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.Trim(scanner.Text(), " ") != "" {
				if string(strings.Trim(scanner.Text(), " ")[0]) != "#" {
					lineData := strings.Split(scanner.Text(), ",")
					if v, e := string2epoch(lineData[0]); e == nil {
						//lineData[0] = strconv.Itoa(int(v))
						//newData = append(newData, lineData)
						label := supp.StringLimit("sample", supp.LabelLength)
						label += supp.StringLimit(strings.Trim(lineData[1], " "), supp.LabelLength)
						label += supp.StringLimit(strings.Trim(lineData[2], " "), supp.LabelLength)
						s0 := &SeriesSample{Stag: label, Sts: v * 1000}
						s1 := &SeriesSample{Stag: label, Sts: (v + 86399) * 1000}

						//if tag, ts, vals, e := ReadSeriesTS(s0, s1, true); e == nil {
						//	fmt.Println(tag, ts, vals)
						//}
						//fmt.Println(DeleteSeriesTS(s0, s1, true))

						if err := DeleteSeriesTS(s0, s1, true); err == nil {
							var testData []SeriesSample
							for i := 3; i < len(lineData); i++ {
								sampleRaw := strings.Split(strings.Trim(lineData[i], " "), "/")
								//fmt.Println(sampleRaw)
								if th, err := strconv.ParseInt(strings.Split(sampleRaw[0], ":")[0], 10, 64); err == nil {
									th *= 3600
									if tm, err := strconv.ParseInt(strings.Split(sampleRaw[0], ":")[1], 10, 64); err == nil {
										tm *= 60
										if val, err := strconv.Atoi(sampleRaw[1]); err == nil {
											newSample := new(SeriesSample)
											newSample.Stag = label
											newSample.Sts = (th + tm + v) * 1000
											newSample.Sval = val
											testData = append(testData, *newSample)
											//fmt.Println(newSample)
											if err := StoreSampleTS(newSample, true); err != nil {
												go func() {
													supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "error writing data",
														[]int{1}, true}
												}()
											}
										}
									}
								}
							}
							if tag, ts, vals, e := ReadSeriesTS(s0, s1, true); e == nil {
								//fmt.Println(tag,ts,vals)
								readData := s0.UnmarshalSliceSS(tag, ts, vals)
								if len(readData) != len(testData) {
									go func() {
										supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "error writing all data",
											[]int{1}, true}
									}()
									//fmt.Println(readData)
									//fmt.Println(testData)
								} else {
									go func() {
										supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "retrieved sample data",
											[]int{1}, true}
									}()
									// entry/flow data is also removed
									labelEntry := supp.StringLimit("entry", supp.LabelLength)
									labelEntry += supp.StringLimit(strings.Trim(lineData[1], " "), supp.LabelLength)
									labelEntry += supp.StringLimit(strings.Trim(lineData[2], " "), supp.LabelLength)
									s0e := &SeriesEntries{Stag: labelEntry, Sts: v * 1000}
									s1e := &SeriesEntries{Stag: labelEntry, Sts: (v + 86399) * 1000}
									if err := DeleteSeriesTS(s0e, s1e, true); err != nil {
										go func() {
											supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "error deleting entry/flow data", []int{1}, true}
										}()
									} else {
										flowCorruptions += 1
									}
									if startup {
										sampleCorruptions += 1
									}
								}
							}
						} else {
							go func() {
								supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "error deleting sample data", []int{1}, true}
							}()
						}
					}
				}
			}
			// newData is an array containing the DBS data with epoch instead of the readable data format as in FORM
			//fmt.Println(newData)

			if err := scanner.Err(); err != nil {
				go func() {
					supp.DLog <- supp.DevData{"storage.RetrieveSampleFromFile", supp.Timestamp(), "error reading .recoverysamples", []int{1}, true}
				}()
			}
		}
		_ = file.Close()
		if startup {
			log.Printf("!!! WARNING DATABASE INTEGRITY CHECK COMPLETED: %v DATA and %v FLOW CORRUPTIONS REMOVED !!!", sampleCorruptions, flowCorruptions)
			_ = os.Remove(SAMPLEFILE)
		}
	}
}

// start database sample values retrieval following format given in .recoverypresence
// this includes also removal of all sample data in the given interval
// the recovery files needs to respect the following format. For each day to be retrieved
// day, space_name, measurement_name, time24h, value
// Example
// January 15 2019, living, morning, 10:30, 15
// Furthermore, recovery files will be rejected of older than MAXAGE hours (TBD)
// Comments in the recovery file need to start with the symbol #
// TODO to be tested it in a real installation
//noinspection GoUnusedParameter
func RetrievePresenceFromFile(startup bool) {
	//fmt.Println("RetrievePresenceFromFile")

	presenceCorruptions := 0

	if fileStat, err := os.Stat(SAMPLEFILE); err == nil {
		if time.Now().Unix()-fileStat.ModTime().Unix() > MAXAGE*3600 && !startup {
			go func() {
				supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "attempted to use an old .recoverypresence", []int{1}, true}
			}()
			return
		}
		if startup {
			log.Println("!!! WARNING PRESENCE DATABASE INTEGRITY CHECK INITIATED !!!")
		}
	} else {
		if startup {
			go func() {
				supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "attempted to use an old .recoverypresence", []int{1}, true}
			}()
			return
		} else {
			log.Println("!!! WARNING PRESENCE DATABASE INTEGRITY IS OK !!!")
			return
		}
	}

	//if fileStat, err := os.Stat(PRESENCEFILE); err == nil {
	//	if time.Now().Unix()-fileStat.ModTime().Unix() > MAXAGE*3600 {
	//		go func() {
	//			supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "attempted to use an old .recoverypresence", []int{1}, true}
	//		}()
	//		return
	//	}
	//} else {
	//	go func() {
	//		supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "attempted to use an old .recoverypresence", []int{1}, true}
	//	}()
	//	return
	//}

	if file, err := os.Open(PRESENCEFILE); err != nil {
		go func() {
			supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "err", []int{1}, true}
		}()
		if startup {
			log.Println("!!! WARNING PRESENCE DATABASE RECOVERY ERROR !!!")
		}
	} else {
		//noinspection GoUnhandledErrorResult
		//defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.Trim(scanner.Text(), " ") != "" {
				if string(strings.Trim(scanner.Text(), " ")[0]) != "#" {
					lineData := strings.Split(scanner.Text(), ",")
					if v, e := string2epoch(lineData[0]); e == nil {
						label := supp.StringLimit("presence", supp.LabelLength)
						label += supp.StringLimit(strings.Trim(lineData[1], " "), supp.LabelLength)
						label += supp.StringLimit(strings.Trim(lineData[2], " "), supp.LabelLength)
						s0 := &SeriesSample{Stag: label, Sts: v * 1000}
						s1 := &SeriesSample{Stag: label, Sts: (v + 86399) * 1000}
						//fmt.Println(s0, s1)
						//fmt.Println("New cycle")
						//if tag, ts, vals, e := ReadSeriesSD(s0, s1, true); e == nil {
						//for _, el := range s0.UnmarshalSliceSS(tag, ts, vals) {
						//	fmt.Println(el)
						//}
						//fmt.Println(DeleteSeriesTS(s0, s1, true))
						if err := DeleteSeriesSD(s0, s1, true); err == nil {
							//for i := 3; i < len(lineData); i++ {
							sampleRaw := strings.Split(strings.Trim(lineData[3], " "), "/")
							//fmt.Println(sampleRaw)
							if th, err := strconv.ParseInt(strings.Split(sampleRaw[0], ":")[0], 10, 64); err == nil {
								th *= 3600
								if tm, err := strconv.ParseInt(strings.Split(sampleRaw[0], ":")[1], 10, 64); err == nil {
									tm *= 60
									if val, err := strconv.Atoi(sampleRaw[1]); err == nil {
										newSample := new(SeriesSample)
										newSample.Stag = label
										newSample.Sts = (th + tm + v) * 1000
										newSample.Sval = val
										//fmt.Println(newSample)
										//fmt.Println(StoreSampleTS(newSample, true))
										if err := StoreSampleSD(newSample, true); err != nil {
											go func() {
												supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "error writing data",
													[]int{1}, true}
											}()
										} else {
											go func() {
												supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "retrieved presence data",
													[]int{1}, true}
											}()
											if startup {
												presenceCorruptions += 1
											}
										}
									}
								}
							}

							//}
							//if tag, ts, vals, e := ReadSeriesSD(s0, s1, true); e == nil {
							//	//fmt.Println(tag, ts, vals)
							//	readData := s0.UnmarshalSliceSS(tag, ts, vals)
							//	for _, el := range readData {
							//		fmt.Println(el)
							//	}
							//}
						} else {
							go func() {
								supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "error deleting data", []int{1}, true}
							}()
						}
					}
				}
			}
			//}

			if err := scanner.Err(); err != nil {
				go func() {
					supp.DLog <- supp.DevData{"storage.RetrievePresenceFromFile", supp.Timestamp(), "error reading .recoverypresence", []int{1}, true}
				}()
			}
		}

		_ = file.Close()
		if startup {
			log.Printf("!!! WARNING PRESENCE DATABASE INTEGRITY CHECK COMPLETED: %v DATA CORRUPTIONS REMOVED !!!", presenceCorruptions)
			_ = os.Remove(SAMPLEFILE)
		}
	}
}
