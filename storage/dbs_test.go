package storage

import (
	"countingserver/support"
	"fmt"
	"testing"
	"time"
)

func Test_Setup(t *testing.T) {
	if err := TimedIntDBSSetUp(true); err != nil {
		t.Fatal(err)
	}
	TimedIntDBSClose()
}

func Test_Marshall(t *testing.T) {

	a := headerData{}
	a.fromRst = uint64(support.Timestamp())
	a.step = 500
	a.lastUpdt = a.fromRst
	a.created = a.fromRst
	b := a.marshall()
	if c, _ := b.unmarshall(); c != a {
		t.Fatal("Wring conversions")
	}
}

func Test_DBS(t *testing.T) {
	if err := TimedIntDBSSetUp(true); err != nil {
		t.Fatal(err)
	}
	//defer TimedIntDBSClose()

	if f, err := SetSeries("test", 2, false); err != nil {
		t.Fatal(err)
	} else {
		if f {
			fmt.Println("Serie definition:", GetDefinition("test"))
		}
	}
	time.Sleep(2 * time.Second)
	ts := support.Timestamp()
	if err := StoreSerieSample("test", ts, 8, false); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	if v, e := ReadSeries("test", ts-200000, ts, false); e != nil {
		t.Fatal(e)
	} else {
		fmt.Println(v)
	}
}
