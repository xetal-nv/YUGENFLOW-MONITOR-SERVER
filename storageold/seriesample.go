package storageold

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
)

// implements sampledata and servers.genericdata for managing data of type "sample"

type SeriesSample struct {
	Stag string `json:"tag"`
	Sts  int64  `json:"ts"`
	Sval int    `json:"val"`
}

func (ss *SeriesSample) SetTag(nm string) {
	ss.Stag = nm
}

func (ss *SeriesSample) SetVal(v ...int) {
	if len(v) == 1 {
		ss.Sval = v[1]
	}
}

func (ss *SeriesSample) SetTs(ts int64) {
	ss.Sts = ts
}

func (ss *SeriesSample) Valid() bool {
	if ss.Sts > 0 {
		return true
	} else {
		return false
	}
}

func (ss *SeriesSample) MarshalSize() int { return 12 }

func (ss *SeriesSample) MarshalSizeModifiers() []int { return []int{0, 0} }

func (ss *SeriesSample) Ts() int64 { return ss.Sts }

func (ss *SeriesSample) Tag() string { return ss.Stag }

func (ss *SeriesSample) Val() int { return ss.Sval }

func (ss *SeriesSample) Marshal() []byte {
	vb := make([]byte, 4)
	binary.LittleEndian.PutUint32(vb, uint32(ss.Sval))
	vts := make([]byte, 8)
	binary.LittleEndian.PutUint64(vts, uint64(ss.Sts))

	return append(vts, vb...)
}

func (ss *SeriesSample) Extract(i interface{}) error {
	if i == nil {
		return errors.New("storage.SeriesSample.Extract: error illegal data received")
	}
	rv := reflect.ValueOf(i)
	defer func() {
		if e := recover(); e != nil {
			_ = ss.Extract(nil)
		}
	}()
	z := SeriesSample{Stag: rv.Field(0).String(), Sts: rv.Field(1).Int(), Sval: int(rv.Field(2).Int())}
	*ss = z
	return nil
}

func (ss *SeriesSample) Unmarshal(c []byte) error {
	if len(c) != 12 {
		return errors.New("storage.SeriesSample.Unmarshal: wrong byte array passed")
	}
	vts := c[0:8]
	vb := c[8:12]
	var val int32
	var ts int64
	buf := bytes.NewReader(vts)
	if err := binary.Read(buf, binary.LittleEndian, &ts); err != nil {
		return errors.New("storage.SeriesSample.Unmarshal: binary.Read failed: " + err.Error())
	}
	buf = bytes.NewReader(vb)
	if err := binary.Read(buf, binary.LittleEndian, &val); err != nil {
		return errors.New("storage.SeriesSample.Unmarshal: binary.Read failed: " + err.Error())
	}
	*ss = SeriesSample{"", ts, int(val)}
	return nil
}

func (ss *SeriesSample) UnmarshalSliceSS(tag string, ts []int64, vals [][]byte) (rt []SampleData) {
	for i, mt := range vals {
		a := new(SeriesSample)
		if e := a.Unmarshal(mt); e == nil {
			//fmt.Println(a.Sts, ts[i])
			a.Stag = tag
			if ts[i] != 0 {
				a.Sts = ts[i]
			}
			rt = append(rt, a)
		}
	}
	return rt
}

func (ss *SeriesSample) UnmarshalSliceNative(tag string, ts []int64, vals [][]byte) (rt []SeriesSample) {
	for i, mt := range vals {
		a := new(SeriesSample)
		if e := a.Unmarshal(mt); e == nil {
			//fmt.Println(a.Sts, ts[i])
			a.Stag = tag
			if ts[i] != 0 {
				a.Sts = ts[i]
			}
			rt = append(rt, *a)
		}
	}
	return rt
}

func SerieSampleDBS(id string, in chan interface{}, rst chan bool, tp string) {

	handlerDBS(id, in, rst, new(SeriesSample), tp)
}
