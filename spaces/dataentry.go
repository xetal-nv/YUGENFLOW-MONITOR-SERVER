package spaces

import (
	"errors"
	"reflect"
)

type dataEntry struct {
	num int   // entry number
	val int   // data received
	ts  int64 // timestamp
}

func (de *dataEntry) Extract(i interface{}) error {
	if i == nil {
		return errors.New("spaces.dataEntry.Extract: error illegal data received")
	}
	rv := reflect.ValueOf(i)
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				_ = de.Extract(nil)
			}
		}
	}()
	z := dataEntry{int(rv.Field(0).Int()), int(rv.Field(1).Int()), rv.Field(2).Int()}
	*de = z
	return nil
}
