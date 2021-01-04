// +build !dev,!debug,script

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// example exporting current data

// generic flow
type FlowWithFlows struct {
	Id         string `json:"id"`
	Variation  int    `json:"-"`
	Netflow    int    `json:"netflow"`
	TsOverflow int64  `json:"overflowTs"`
	Reversed   bool   `json:"-"`
	FlowIn     int    `json:"in"`
	FlowOut    int    `json:"out"`
}

// entry flow data model used for database storage
type EntryStateWithFlows struct {
	Id         string                   `json:"id"`
	Ts         int64                    `json:"Ts"`
	Variation  int                      `json:"-"`
	Netflow    int                      `json:"netflow"`
	TsOverflow int64                    `json:"overflowTs"`
	FlowIn     int                      `json:"in"`
	FlowOut    int                      `json:"out"`
	State      bool                     `json:"-"`
	Reversed   bool                     `json:"-"`
	Flows      map[string]FlowWithFlows `json:"flows"`
}

type MeasurementSample struct {
	Qualifier      string                         `json:"qualifier"`
	Space          string                         `json:"space"`
	Ts             int64                          `json:"timestamp"`
	Count          float64                        `json:"count"`
	FlowIn         int                            `json:"in"`
	FlowOut        int                            `json:"out"`
	StartTimeFlows int64                          `json:"startTimeFlows"`
	TsOverflow     int64                          `json:"overflowTs"`
	Flows          map[string]EntryStateWithFlows `json:"flows"`
}

func main() {
	if len(os.Args) > 1 {
		var receivedData MeasurementSample
		str := strings.Replace(os.Args[1], "'", "\"", -1)
		if file, err := os.OpenFile("exportedData.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err == nil {
			_, _ = file.WriteString(str + "\n")
			defer file.Close()
			if err := json.Unmarshal([]byte(str), &receivedData); err == nil {
				_, _ = file.WriteString("Space " + receivedData.Space + " has counter " + receivedData.Qualifier +
					" equal to " + fmt.Sprintf("%f", receivedData.Count) + " at time " + strconv.Itoa(int(receivedData.Ts)) + "\n")
			}
		}
	}
}
