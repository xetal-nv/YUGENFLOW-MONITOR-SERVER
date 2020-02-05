package storage

import (
	"gateserver/support"
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

// Convert SerieEntries into the json friendlier JsonSeriesEntries
func (ret *JsonSeriesEntries) ExpandEntries(ss SerieEntries) {
	// fmt.Println(ss)
	if ss.Stag != "" {
		r, _ := regexp.Compile("_+")
		tmp := r.ReplaceAllString(ss.Stag, "_")
		name := support.StringLimit(strings.Split(tmp, "_")[1], support.LabelLength)
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
	// TODO there is sometime an issue here as it failes to extract
	// fmt.Println(i)
	tmp := SerieEntries{}
	err = tmp.Extract(i)
	// fmt.Println(tmp)
	if err == nil {
		ss.ExpandEntries(tmp)
	}
	// fmt.Println(err, *ss)
	return
}
