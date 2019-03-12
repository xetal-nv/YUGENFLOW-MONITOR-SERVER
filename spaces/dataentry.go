package spaces

import (
	"errors"
	"reflect"
	"strconv"
)

type dataEntry struct {
	id  string // entry id as string to support entry data in the entire comunication pipe
	ts  int64  // timestamp
	val int    // data received
}

type spaceEntries struct {
	id      int               // entry id
	ts      int64             // timestap for the cumulative value
	val     int               // cumulative value
	entries map[int]dataEntry // cumulative value per entry
}

func (de *dataEntry) Extract(i interface{}) error {
	if i == nil {
		return errors.New("spaces.dataEntry.Extract: error illegal data received")
	}
	rv := reflect.ValueOf(i)
	defer func() {
		if e := recover(); e != nil {
			_ = de.Extract(nil)
		}
	}()
	z := dataEntry{id: strconv.Itoa(int(rv.Field(0).Int())), ts: rv.Field(2).Int(), val: int(rv.Field(1).Int())}
	*de = z
	return nil
}
