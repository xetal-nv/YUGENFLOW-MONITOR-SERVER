package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
)

//type serieSample struct {
//	ts  int64
//	val int
//}

type SerieSample struct {
	tag string
	ts  int64
	val int
}

func (ss *SerieSample) MarshalSize() int { return 12 }

func (ss *SerieSample) Ts() int64 { return ss.ts }

func (ss *SerieSample) Tag() string { return ss.tag }

func (ss *SerieSample) Val() int { return ss.val }

func (ss *SerieSample) Marshal() []byte {
	vb := make([]byte, 4)
	binary.LittleEndian.PutUint32(vb, uint32(ss.val))
	vts := make([]byte, 8)
	binary.LittleEndian.PutUint64(vts, uint64(ss.ts))

	return append(vts, vb...)
}

func (ss *SerieSample) Extract(i interface{}) error {
	if i == nil {
		return errors.New("storage.SerieSample.Extract: error illegal data received")
	}
	rv := reflect.ValueOf(i)
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				_ = ss.Extract(nil)
			}
		}
	}()
	z := SerieSample{rv.Field(0).String(), rv.Field(1).Int(), int(rv.Field(2).Int())}
	*ss = z
	return nil
}

func (ss *SerieSample) Unmarshal(c []byte) error {
	if len(c) != 12 {
		return errors.New("storage.SerieSample.Unmarshal: wrong byte array passed")
	}
	vts := c[0:8]
	vb := c[8:12]
	var val int32
	var ts int64
	buf := bytes.NewReader(vts)
	if err := binary.Read(buf, binary.LittleEndian, &ts); err != nil {
		return errors.New("storage.SerieSample.Unmarshal: binary.Read failed: " + err.Error())
	}
	buf = bytes.NewReader(vb)
	if err := binary.Read(buf, binary.LittleEndian, &val); err != nil {
		return errors.New("storage.SerieSample.Unmarshal: binary.Read failed: " + err.Error())
	}
	*ss = SerieSample{"", ts, int(val)}
	return nil
}

func UnmarshalSlice(tag string, ts []int64, vals [][]byte) []SerieSample {
	var rt []SerieSample
	for i, mt := range vals {
		a := new(SerieSample)
		if e := a.Unmarshal(mt); e == nil {
			a.tag = tag
			a.ts = ts[i]
			rt = append(rt, *a)
		}
	}
	return rt
}

func SerieSampleDBS(id string, in chan interface{}, rst chan bool) {

	handlerDBS(id, in, rst, new(SerieSample))
}
