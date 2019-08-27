package spaces

import (
	"errors"
	"reflect"
	"strconv"
)

type DataEntry struct {
	id           string // entry Id as string to support entry data in the entire communication pipe
	Ts           int64  // timestamp
	NetFlow      int    // net flow
	PositiveFlow int    // counter positive (entries) transactions
	NegativeFlow int    // counter negative (exits) transactions
}

type spaceEntries struct {
	id      int               // entry Id
	ts      int64             // timestamp for the cumulative value
	netflow int               // cumulative net flow
	entries map[int]DataEntry // data per entry
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
	//z := DataEntry{id: strconv.Itoa(int(rv.Field(0).Int())), Ts: rv.Field(1).Int(), NetFlow: int(rv.Field(2).Int())}
	z := DataEntry{id: strconv.Itoa(int(rv.Field(0).Int())), Ts: rv.Field(1).Int(), NetFlow: int(rv.Field(2).Int()),
		PositiveFlow: int(rv.Field(3).Int()), NegativeFlow: int(rv.Field(4).Int())}
	*de = z
	return nil
}
