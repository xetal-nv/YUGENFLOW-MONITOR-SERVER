package storage

import (
	"countingserver/support"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

type SerieEntries struct {
	Stag string  `json:"Stag"`
	Sts  int64   `json:"Sts"`
	Sval [][]int `json:"Sval"`
}

func (ss *SerieEntries) SetTag(nm string) {
	ss.Stag = nm
}

// TODO
func (ss *SerieEntries) SetVal(v ...int) {
	// this does nothing
}

func (ss *SerieEntries) SetTs(ts int64) {
	ss.Sts = ts
}

func (ss *SerieEntries) Valid() bool {
	if ss.Sts > 0 && len(ss.Sval) > 0 {
		return true
	} else {
		return false
	}
}

// it needs the variable read from the DBS with md = 12, fs = 8
func (ss *SerieEntries) MarshalSize() int { return 0 }

// returns the recurrent data size (Sval) and the offset data size for database read
func (ss *SerieEntries) MarshalSizeModifiers() []int { return []int{8, support.LabelLength + 8} }

func (ss *SerieEntries) Ts() int64 { return ss.Sts }

func (ss *SerieEntries) Tag() string { return ss.Stag }

func (ss *SerieEntries) Val() [][]int { return ss.Sval }

// FORMAT fiels:N_units:unit_in_bytes
// FORMAT (LENGTH:2:1, TAG:support.LabelLength:1, TS:1:1, VAL:(LENGTH):8
func (ss *SerieEntries) Marshal() (rt []byte) {
	ll := len(ss.Sval)
	msg := make([]byte, 2)
	binary.LittleEndian.PutUint16(msg, uint16(ll))
	msg = append(msg, []byte(ss.Stag)...)
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
// see spaces.passData in samplers.go
func (ss *SerieEntries) Extract(i interface{}) (err error) {
	err = nil
	if i == nil {
		err = errors.New("storage.SerieSample.Extract: error illegal data received")
		return
	}
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				fmt.Println(e)
				_ = ss.Extract(nil)
			}
		}
	}()
	rv := reflect.ValueOf(i)
	z := SerieEntries{Stag: rv.Field(0).String(), Sts: rv.Field(1).Int()}
	ll := int(rv.Field(2).Int())
	for j := 0; j < ll; j++ {
		v := []int{int(rv.Field(3).Index(j).Index(0).Int()),
			int(rv.Field(3).Index(j).Index(1).Int())}
		z.Sval = append(z.Sval, v)
	}
	*ss = z
	return
}

// FORMAT fiels:N_units:unit_in_bytes
// FORMAT (LENGTH:2:1, TAG:support.LabelLength:1, TS:1:1, VAL:(LENGTH):8
func (ss *SerieEntries) Unmarshal(c []byte) error {
	offsets := ss.MarshalSizeModifiers()
	if len(c[2:]) != (int(binary.LittleEndian.Uint16(c[0:2]))*offsets[0] + offsets[1]) {
		return errors.New("storage.SerieEntries.Unmarshal illegale code size")
	}
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				fmt.Println(e)
				_ = ss.Unmarshal(c[0:2])
			}
		}
	}()

	ss.Stag = string(c[2:(2 + support.LabelLength)])
	ss.Sts = int64(binary.LittleEndian.Uint64(c[(2 + support.LabelLength):(2 + offsets[1])]))
	for n := (2 + offsets[1]); n < len(c); n += offsets[0] {
		val := []int{int(binary.LittleEndian.Uint32(c[n : n+4])), int(binary.LittleEndian.Uint32(c[n+4 : n+8]))}
		ss.Sval = append(ss.Sval, val)
	}
	return nil
}

//func UnmarshalSliceSE(Stag string, Sts []int64, vals [][]byte) (rt []SerieEntries) { // TBD
//	for i, mt := range vals {
//		a := new(SerieEntries)
//		if e := a.Unmarshal(mt); e == nil {
//			a.Stag = Stag
//			a.Sts = Sts[i]
//			rt = append(rt, *a)
//		}
//	}
//	return rt
//}

func SeriesEntryDBS(id string, in chan interface{}, rst chan bool) {

	handlerDBS(id, in, rst, new(SerieEntries))
}
