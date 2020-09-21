package storage

import (
	"bytes"
	"encoding/binary"
	"gateserver/supp"
	"reflect"
	"regexp"
	"strings"

	"errors"
)

// implements sampledata and servers.genericdata for managing data of type "entry"

type SeriesEntries struct {
	Stag string  `json:"tag"`
	Sts  int64   `json:"ts"`
	Sval [][]int `json:"val"`
}

func (ss *SeriesEntries) SetTag(nm string) {
	ss.Stag = nm
}

//noinspection GoUnusedParameter
func (ss *SeriesEntries) SetVal(v ...int) {
	// this does nothing
}

func (ss *SeriesEntries) SetTs(ts int64) {
	ss.Sts = ts
}

func (ss *SeriesEntries) Valid() bool {
	if ss.Sts > 0 && len(ss.Sval) > 0 {
		return true
	} else {
		return false
	}
}

// it needs the variable read from the DBS with md = 12, fs = 8
func (ss *SeriesEntries) MarshalSize() int { return 0 }

// returns the recurrent data size (Sval) and the offset data size for database read
func (ss *SeriesEntries) MarshalSizeModifiers() []int { return []int{8, 8} }

func (ss *SeriesEntries) Ts() int64 { return ss.Sts }

func (ss *SeriesEntries) Tag() string { return ss.Stag }

func (ss *SeriesEntries) Val() [][]int { return ss.Sval }

// FORMAT fiels:N_units:unit_in_bytes
// FORMAT (LENGTH:2:1, TS:1:1, VAL:(LENGTH):8
func (ss *SeriesEntries) Marshal() (rt []byte) {
	ll := len(ss.Sval)
	msg := make([]byte, 2)
	binary.LittleEndian.PutUint16(msg, uint16(ll))
	hp := make([]byte, 8)
	binary.LittleEndian.PutUint64(hp, uint64(ss.Sts))
	msg = append(msg, hp...)
	for _, v := range ss.Sval {
		if len(v) != 2 {
			return nil
		}
		id := make([]byte, 4)
		val := make([]byte, 4)
		binary.LittleEndian.PutUint32(id, uint32(v[0]))
		binary.LittleEndian.PutUint32(val, uint32(v[1]))
		msg = append(msg, id...)
		msg = append(msg, val...)
	}
	rt = make([]byte, len(msg))
	copy(rt, msg)
	return
}

// Extract assumes that after Sts there is a extra int that gives the length of the 2s array
// This function is to extract the database series data structure (not the full version)
// see spaces.passData in samplers.go
func (ss *SeriesEntries) Extract(i interface{}) (err error) {
	// fmt.Println("this one")
	err = nil
	if i == nil {
		err = errors.New("storage.SeriesSample.Extract: error illegal data received")
		return
	}
	defer func() {
		if e := recover(); e != nil {
			_ = ss.Extract(nil)
		}
	}()
	rv := reflect.ValueOf(i)
	// fmt.Println(rv)
	//fmt.Println("Extract series: ", rv)
	z := SeriesEntries{Stag: rv.Field(0).String(), Sts: rv.Field(1).Int()}
	// fmt.Println(z)
	var entries [][]int
	var name string
	if rv.Field(0).String() != "" {
		// fmt.Println("here")
		r, _ := regexp.Compile("_+")
		tmp := r.ReplaceAllString(rv.Field(0).String(), "_")
		name = supp.StringLimit(strings.Split(tmp, "_")[1], supp.LabelLength)
		// fmt.Println(name)
		// fmt.Println(SpaceInfo)
		for range SpaceInfo[name] {
			entries = append(entries, []int{0, 0})
		}
		// fmt.Println(entries)
	}
	ll := int(rv.Field(2).Int())
	for j := 0; j < ll; j++ {
		id := rv.Field(3).Index(j).Index(0).Int()
		pos := int(rv.Field(3).Index(j).Index(2).Int())
		neg := int(rv.Field(3).Index(j).Index(3).Int())
		// fmt.Println(pos, neg)
		//v := []int{int(rv.Field(3).Index(j).Index(0).Int()),
		//	int(rv.Field(3).Index(j).Index(1).Int())}
		//z.Sval = append(z.Sval, v)
		// fmt.Println(entries, int(id))
		if entries != nil {
			index := supp.SliceIndex(len(SpaceInfo[name]), func(i int) bool { return SpaceInfo[name][i] == int(id) })
			// fmt.Println(id, supp.SliceIndex(len(SpaceInfo[name]), func(i int) bool { return SpaceInfo[name][i] == int(id) }))
			// if int(id) < len(entries) { // the if is redundant
			entries[index] = []int{pos, neg}
			// }
		}
		// fmt.Println(entries)
	}
	z.Sval = entries
	*ss = z
	// testing
	//fmt.Println(z)
	//tmp := JsonSeriesEntries{}
	//tmp.ExpandEntries(*ss)
	//fmt.Println(tmp)
	return
}

// ExtractForRecovery assumes that after Sts there is a extra int that gives the length of the 2s array
// This function is to extract the full series data structure (not the database version)
// see spaces.passData in samplers.go
func (ss *SeriesEntries) ExtractForRecovery(i interface{}) (err error) {
	err = nil
	if i == nil {
		err = errors.New("storage.SeriesSample.Extract: error illegal data received")
		return
	}
	defer func() {
		if e := recover(); e != nil {
			_ = ss.Extract(nil)
		}
	}()
	rv := reflect.ValueOf(i)
	z := SeriesEntries{Stag: rv.Field(0).String(), Sts: rv.Field(1).Int()}
	ll := int(rv.Field(2).Int())
	for j := 0; j < ll; j++ {
		v := []int{int(rv.Field(3).Index(j).Index(0).Int()), int(rv.Field(3).Index(j).Index(1).Int()),
			int(rv.Field(3).Index(j).Index(2).Int()), int(rv.Field(3).Index(j).Index(3).Int())}
		z.Sval = append(z.Sval, v)
	}
	*ss = z
	return
}

// FORMAT fields:N_units:unit_in_bytes
// FORMAT (LENGTH:2:1, TS:1:1, VAL:(LENGTH):8
func (ss *SeriesEntries) Unmarshal(c []byte) error {
	offsets := ss.MarshalSizeModifiers()
	if len(c[2:]) != (int(binary.LittleEndian.Uint16(c[0:2]))*offsets[0] + offsets[1]) {
		return errors.New("storage.SeriesEntries.Unmarshal illegal code size ")
	}
	defer func() {
		if e := recover(); e != nil {
			_ = ss.Unmarshal(c[0:2])
		}
	}()

	ss.Sts = int64(binary.LittleEndian.Uint64(c[2:(2 + offsets[1])]))
	for n := 2 + offsets[1]; n < len(c); n += offsets[0] {
		v1 := c[n : n+4]
		v2 := c[n+4 : n+8]
		var g, gv int32
		buf := bytes.NewReader(v1)
		if err := binary.Read(buf, binary.LittleEndian, &g); err != nil {
			return errors.New("storage.SeriesSample.Unmarshal: binary.Read failed: " + err.Error())
		}
		buf = bytes.NewReader(v2)
		if err := binary.Read(buf, binary.LittleEndian, &gv); err != nil {
			return errors.New("storage.SeriesSample.Unmarshal: binary.Read failed: " + err.Error())
		}
		ss.Sval = append(ss.Sval, []int{int(g), int(gv)})
	}
	return nil
}

func (ss *SeriesEntries) UnmarshalSliceSS(tag string, ts []int64, vals [][]byte) (rt []SampleData) {
	for i, mt := range vals {
		a := new(SeriesEntries)
		//fmt.Println(mt)
		//fmt.Println(a.Unmarshal(mt))
		if e := a.Unmarshal(mt); e == nil {
			//fmt.Println(mt, a)
			a.Stag = tag
			a.Sts = ts[i]
			rt = append(rt, a)
		}
	}
	return rt
}

func SeriesEntryDBS(id string, in chan interface{}, rst chan bool, tp string) {

	handlerDBS(id, in, rst, new(SeriesEntries), tp)
}
