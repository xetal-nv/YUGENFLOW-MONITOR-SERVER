package spaces

import (
	"errors"
	"reflect"
	"strconv"
)

type DataEntry struct {
	id  string // entry Id as string to support entry data in the entire communication pipe
	Ts  int64  // timestamp
	Val int    // data received
}

type spaceEntries struct {
	id      int               // entry Id
	ts      int64             // timestamp for the cumulative value
	val     int               // cumulative value
	entries map[int]DataEntry // cumulative value per entry
}

// extract a DataEntry value from a generic interface{} if possible
func (de *DataEntry) Extract(i interface{}) error {
	if i == nil {
		return errors.New("spaces.DataEntry.Extract: error illegal data received")
	}
	rv := reflect.ValueOf(i)
	defer func() {
		if e := recover(); e != nil {
			_ = de.Extract(nil)
		}
	}()
	z := DataEntry{id: strconv.Itoa(int(rv.Field(0).Int())), Ts: rv.Field(2).Int(), Val: int(rv.Field(1).Int())}
	*de = z
	return nil
}
