package apiManager

import "gateserver/dataformats"

type JsonMeasurement struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Interval int    `json:"interval"`
}

type JsonDevices struct {
	Id        int  `json:"deviceId"`
	Reversed  bool `json:"reversed"`
	Suspected bool `json:"suspected"`
	Disabled  bool `json:"disabled"`
}

type JsonGate struct {
	Id       string        `json:"gateName"`
	Devices  []JsonDevices `json:"devices"`
	Reversed bool          `json:"reversed"`
}

type JsonEntry struct {
	Id       string     `json:"entryName"`
	Gates    []JsonGate `json:"gates"`
	Reversed bool       `json:"reversed"`
}

type JsonSpace struct {
	Id      string      `json:"spacename"`
	Entries []JsonEntry `json:"entries"`
}

type JsonConnectedDevice struct {
	Mac    string `json:"mac"`
	Active bool   `json:"active"`
}

type JsonInvalidDevice struct {
	Mac string `json:"mac"`
	Ts  int64  `json:"timestamp"`
}

type JsonData struct {
	Space   string                                     `json:"space"`
	Type    string                                     `json:"type"`
	Results map[string][]dataformats.MeasurementSample `json:"measurements"`
}

type JsonPresence struct {
	Space    string `json:"space"`
	Presence bool   `json:"presence"`
}

type JsonCmdRt struct {
	Answer string `json:"answer"`
	Error  string `json:"error"`
}
