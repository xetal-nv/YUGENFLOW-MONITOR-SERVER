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

func Test_SerieSample(t *testing.T) {

	a := headerdata{}
	a.fromRst = uint64(support.Timestamp())
	a.step = 500
	a.lastUpdt = a.fromRst
	a.created = a.fromRst
	b := a.Marshal()
	var c headerdata
	if e := c.Unmarshal(b); e != nil || c != a {
		t.Fatal("Wring conversions 1:", e)
	}

	x := SerieSample{"test", int64(a.fromRst), 8}
	y := x.Marshal()
	z := new(SerieSample)

	if e := z.Unmarshal(y); e != nil {
		t.Fatal("Wring conversions 1:", e)
	}
	fmt.Println(*z)

}

func Test_SerieEntries(t *testing.T) {
	var a [][]int
	a = append(a, []int{1, 2})
	a = append(a, []int{4, 5})
	a = append(a, []int{7, 8})
	b := SerieEntries{"all", 123, a}
	fmt.Println(b)
	fmt.Println(b.Marshal())

	r := func(a interface{}) interface{} {
		return a
	}
	bb := struct {
		tag string
		ts  int64
		ll  int
		val [][]int
	}{b.tag, b.ts, 3, b.val}
	c := new(SerieEntries)
	d := r(bb)
	if err := c.Extract(d); err != nil {
		t.Fatal(err)
	}
	fmt.Println("extract:", *c)
	e := new(SerieEntries)
	if err := e.Unmarshal(b.Marshal()); err != nil {
		t.Fatal(err)
	}
	fmt.Println("unmarshal:", *e)
}

func Test_DBSraw(t *testing.T) {
	if err := TimedIntDBSSetUp(true); err != nil {
		t.Fatal(err)
	}
	defer TimedIntDBSClose()

	if e := update([]byte{2, 0, 23, 44, 44, 56}, []byte{34}, *currentDB, true); e == nil {
		val, err := readVar16([]byte{34}, 2, 0, *currentDB)
		fmt.Println(val, err)
	}
}

func Test_DBS(t *testing.T) {
	if err := TimedIntDBSSetUp(true); err != nil {
		t.Fatal(err)
	}
	defer TimedIntDBSClose()

	if f, err := SetSeries("test", 2, false); err != nil {
		t.Fatal(err)
	} else {
		if f {
			fmt.Println("Serie definition:", GetDefinition("test"))
		}
	}
	time.Sleep(2 * time.Second)
	ts := support.Timestamp()
	ts = support.Timestamp()
	a := SerieSample{"test", ts, 11}
	if err := StoreSample(&a, false); err != nil {
		t.Fatal(err)
	}
	s0 := SerieSample{"test", ts - 500000, 8}
	s1 := SerieSample{"test", ts + 1000, 8}
	if tag, ts, vals, e := ReadSerie(&s0, &s1, false); e != nil {
		t.Fatal(e)
	} else {
		fmt.Println(UnmarshalSliceSS(tag, ts, vals))
	}
}
