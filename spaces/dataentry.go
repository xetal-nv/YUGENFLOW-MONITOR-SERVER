package spaces

import (
	"errors"
	"reflect"
)

type dataOneEntry struct {
	num int   // entry number
	val int   // data received
	ts  int64 // timestamp
}

type dataEntry struct {
	val int   // data received
	ts  int64 // timestamp
}

type spaceEntries struct {
	ts      int64             // timestap for the cumulative value
	val     int               // cumulative value
	entries map[int]dataEntry // cumulative value per entry
}

func (de *dataOneEntry) Extract(i interface{}) error {
	if i == nil {
		return errors.New("spaces.dataOneEntry.Extract: error illegal data received")
	}
	rv := reflect.ValueOf(i)
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				_ = de.Extract(nil)
			}
		}
	}()
	z := dataOneEntry{int(rv.Field(0).Int()), int(rv.Field(1).Int()), rv.Field(2).Int()}
	*de = z
	return nil
}
