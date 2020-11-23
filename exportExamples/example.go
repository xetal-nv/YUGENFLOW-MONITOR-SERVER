package exportExamples

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// generic flow
type Flow struct {
	Id      string `json:"id"`
	Netflow int    `json:"netflow"`
}

// entry flow data model used for database storage
type EntryState struct {
	Id       string          `json:"id"`
	Ts       int64           `json:"Ts"`
	Count    int             `json:"netflow"`
	State    bool            `json:"-"`
	Reversed bool            `json:"reversed"`
	Flows    map[string]Flow `json:"flows"`
}

type MeasurementSample struct {
	Qualifier string                `json:"qualifier"`
	Space     string                `json:"space"`
	Ts        int64                 `json:"timestamp"`
	Val       float64               `json:"value"`
	Flows     map[string]EntryState `json:"flows"`
}

func _example() {
	if len(os.Args) > 1 {
		var receivedData MeasurementSample
		str := strings.Replace(os.Args[1], "'", "\"", -1)
		if err := json.Unmarshal([]byte(str), &receivedData); err == nil {
			if file, err := os.OpenFile("exportedData.txt", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				defer file.Close()
				_, _ = file.WriteString("Space " + receivedData.Space + " has counter " + receivedData.Qualifier +
					" equal to " + fmt.Sprintf("%f", receivedData.Val) + " at time " + strconv.Itoa(int(receivedData.Ts)) + "\n")
			}
		}
	}
}
