package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
)

type SerieEntries struct {
	tag string
	ts  int64
	val int
}

func (ss *SerieEntries) MarshalSize() int { return 0 } // TBD

func (ss *SerieEntries) Ts() int64 { return ss.ts }

func (ss *SerieEntries) Tag() string { return ss.tag }

func (ss *SerieEntries) Val() int { return 0 } // TBD

func (ss *SerieEntries) Marshal() []byte { // TBD
	vb := make([]byte, 4)
	binary.LittleEndian.PutUint32(vb, uint32(ss.val))
	vts := make([]byte, 8)
	binary.LittleEndian.PutUint64(vts, uint64(ss.ts))

	return append(vts, vb...)
}

func (ss *SerieEntries) Extract(i interface{}) error { // TBD
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
	z := SerieEntries{rv.Field(0).String(), rv.Field(1).Int(), int(rv.Field(2).Int())}
	*ss = z
	return nil
}

func (ss *SerieEntries) Unmarshal(c []byte) error { // TBS
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
	*ss = SerieEntries{"", ts, int(val)}
	return nil
}

func UnmarshalSliceSE(tag string, ts []int64, vals [][]byte) (rt []SerieEntries) { // TBD
	for i, mt := range vals {
		a := new(SerieEntries)
		if e := a.Unmarshal(mt); e == nil {
			a.tag = tag
			a.ts = ts[i]
			rt = append(rt, *a)
		}
	}
	return rt
}

func SeriesEntryDBS(id string, in chan interface{}, rst chan bool) {

	handlerDBS(id, in, rst, new(SerieEntries))
}
