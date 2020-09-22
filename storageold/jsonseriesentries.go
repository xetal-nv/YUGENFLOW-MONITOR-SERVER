package storageold

import (
	"gateserver/supp"
	"regexp"
	"strings"
)

// This type is used to render SeriesEntry more readable in json

type JsonSingleEntry struct {
	Id      int `json:"id"`
	Netflow int `json:"netflow"`
	In      int `json:"in"`
	Out     int `json:"out"`
}

type JsonSeriesEntries struct {
	Stag string            `json:"tag"`
	Sts  int64             `json:"ts"`
	Sval []JsonSingleEntry `json:"entries"`
}

type JsonCompleteData struct {
	Sts         int64             `json:"ts"`
	AvgPresence int               `json:"avgPresence"`
	Corrupted   bool              `json:"corruptedData"`
	Sval        []JsonSingleEntry `json:"totalEntries"`
}

type JsonCompleteReport struct {
	Stag string             `json:"tag"`
	Meas string             `json:"measurement"`
	Data []JsonCompleteData `json:"data"`
}

// Convert SeriesEntries into the json friendlier JsonSeriesEntries
func (ret *JsonSeriesEntries) ExpandEntries(ss SeriesEntries) {
	if ss.Stag != "" {
		r, _ := regexp.Compile("_+")
		tmp := r.ReplaceAllString(ss.Stag, "_")
		name := supp.StringLimit(strings.Split(tmp, "_")[1], supp.LabelLength)
		//fmt.Println(SpaceInfo[name])
		ret.Stag = ss.Stag
		ret.Sts = ss.Sts
		// SpaceInfo[name] and ss arte ordered in the same way per construction
		for j, i := range SpaceInfo[name] {
			// if i < len(ss.Sval) {
			ret.Sval = append(ret.Sval, JsonSingleEntry{Id: i, In: ss.Sval[j][0], Out: ss.Sval[j][1], Netflow: ss.Sval[j][0] + ss.Sval[j][1]})
			// }
		}
		//for _ = range SpaceInfo[name] {
		//	ret.Sval = append(ret.Sval, []int{0, 0})
		//}
	}
	return
}

func (ss *JsonSeriesEntries) SetTag(nm string) {
	ss.Stag = nm
}

//noinspection GoUnusedParameter
func (ss *JsonSeriesEntries) SetVal(v ...int) {
	// this does nothing
}

func (ss *JsonSeriesEntries) SetTs(ts int64) {
	ss.Sts = ts
}

func (ss *JsonSeriesEntries) Extract(i interface{}) (err error) {
	tmp := SeriesEntries{}
	err = tmp.Extract(i)
	if err == nil {
		ss.ExpandEntries(tmp)
	}
	return
}