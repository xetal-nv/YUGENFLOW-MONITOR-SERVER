package sensormodels

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// DO NOT USE IN PRODUCTION !!!
// ONLY DEVELOPER LICENCE AVAILABLE, MAX 100 REQUESTS PER MONTH.
// FOR COMMERCIAL USE SEE https://getfestivo.com/

type V1 struct {
	Url string
	Key string
}

func NewV1(key string) *V1 {
	v1 := &V1{
		Key: key,
		Url: "https://getfestivo.com/v1/holidays?",
	}
	return v1
}

type Query struct {
	Key     string `json:"api_key"`
	Country string `json:"country"`
	Year    string `json:"year"`
}

type Holiday struct {
	Date   string `json:"date"`
	Start  string `json:"start"`
	End    string `json:"end"`
	Type   string `json:"type"`
	Public bool   `json:"public"`
}

type HolidaysQuery struct {
	Query    Query     `json:"Query"`
	Holidays []Holiday `json:"holidays"`
}

type Response struct {
	Error    bool          `json:"error"`
	Holidays HolidaysQuery `json:"holidays"`
}

//func (v1 *V1) GetHolidays(args map[string]interface{}) (map[string]interface{}, error) {
func (v1 *V1) GetHolidays(args map[string]interface{}) (rp Response, err error) {
	//var data map[string]interface{}

	if _, ok := args["key"]; !ok {
		args["key"] = v1.Key
	}
	params := url.Values{}
	for k, v := range args {
		params.Add(k, v.(string))
	}
	resp, err := http.Get(v1.Url + params.Encode())
	if err != nil {
		return Response{}, err
	}

	//noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	//noinspection GoUnhandledErrorResult
	json.Unmarshal([]byte(string(body)), &rp)
	if resp.StatusCode != 200 {
		err = errors.New("unknown error")
	}

	return rp, err
}

func extractHolidays(country, year string) Response {
	api := NewV1("b065ab3d-11bd-4681-ba3d-2a6f6501cb8c")

	holidays, err := api.GetHolidays(map[string]interface{}{
		// Required
		"api_key": "b065ab3d-11bd-4681-ba3d-2a6f6501cb8c",
		"country": country,
		"year":    year,
		// Optional
		// "month":    "7",
		// "day":      "4",
		// "previous": "true",
		// "upcoming": "true",
		// "public":   "true",
		// "pretty":   "true",
	})

	if err != nil {
		fmt.Println("Error connecting to to the Holiday API:", err)
	}

	//fmt.Println("%#v\n", holidays)

	return holidays
}
