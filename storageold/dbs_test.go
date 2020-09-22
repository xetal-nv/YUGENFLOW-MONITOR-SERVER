package storageold

import (
	"fmt"
	"gateserver/supp"
	"testing"
	"time"
)

func Test_Setup(t *testing.T) {
	if err := TimedIntDBSSetUp("", true); err != nil {
		t.Fatal(err)
	}
	TimedIntDBSClose()
}

func Test_Test_SerieSampleMU(t *testing.T) {
	supp.LabelLength = 8
	b := SeriesSample{"entry___noname__current_", 123, -13}
	fmt.Println(b)
	c := b.Marshal()
	fmt.Println(c, len(c))
	d := new(SeriesSample)
	if e := d.Unmarshal(c); e == nil {
		fmt.Println(*d)
	} else {
		fmt.Println(e)
	}
}

func Test_Test_SerieEntriesMU(t *testing.T) {
	var a [][]int
	supp.LabelLength = 8
	a = append(a, []int{0, -1})
	a = append(a, []int{4, 5})
	a = append(a, []int{7, -8})
	a = append(a, []int{6, 3})
	b := SeriesEntries{"entry___noname__current_", 123, a}
	fmt.Println(b)
	c := b.Marshal()
	fmt.Println(c, len(c))
	d := new(SeriesEntries)
	if e := d.Unmarshal(c); e == nil {
		fmt.Println(*d)
	} else {
		fmt.Println(e)
	}
}

func Test_SerieSample(t *testing.T) {

	a := HeaderData{}
	a.fromRst = uint64(supp.Timestamp())
	a.step = 500
	a.lastUpdate = a.fromRst
	a.created = a.fromRst
	b := a.Marshal()
	var c HeaderData
	if e := c.Unmarshal(b); e != nil || c != a {
		t.Fatal("Wring conversions 1:", e)
	}

	x := SeriesSample{"test", int64(a.fromRst), 8}
	y := x.Marshal()
	z := new(SeriesSample)

	if e := z.Unmarshal(y); e != nil {
		t.Fatal("Wring conversions 1:", e)
	}
	fmt.Println(*z)

}

func Test_SerieEntries(t *testing.T) {
	supp.LabelLength = 8
	if err := TimedIntDBSSetUp("", true); err != nil {
		t.Fatal(err)
	}
	defer TimedIntDBSClose()
	var a [][]int
	a = append(a, []int{1, 2})
	a = append(a, []int{4, 5})
	a = append(a, []int{7, 8})
	a = append(a, []int{6, 3})
	b := SeriesEntries{supp.StringLimit("entry___notame__current_", supp.LabelLength), 123, a}
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
	}{b.Stag, b.Sts, len(b.Sval), b.Sval}
	c := new(SeriesEntries)
	d := r(bb)
	if err := c.Extract(d); err != nil {
		t.Fatal(err)
	}
	fmt.Println("extract:", *c)
	e := new(SeriesEntries)
	if err := e.Unmarshal(b.Marshal()); err != nil {
		t.Fatal(err)
	}
	fmt.Println("unmarshal:", *e)

	if e := update(b.Marshal(), []byte{34}, *currentDB, true); e == nil {
		//offset := b.MarshalSizeModifiers()
		val, err := read([]byte{34}, b.MarshalSize(), b.MarshalSizeModifiers(), *currentDB)
		if err != nil {
			fmt.Println(err)
		} else {
			e := new(SeriesEntries)
			_ = e.Unmarshal(val)
			fmt.Println("From DBS:", *e)
		}
	} else {
		fmt.Println(e)
	}

	if f, err := SetSeries(supp.StringLimit("entry___notame__current_", supp.LabelLength), 2, true); err != nil {
		t.Fatal(err)
	} else {
		if f {
			fmt.Println("Serie definition:", GetDefinition(supp.StringLimit("entry___notame__current_", supp.LabelLength)))
		}
	}

	if err := StoreSampleTS(&b, true, false); err != nil {
		t.Fatal(err)
	}
}

func Test_DBSraw(t *testing.T) {
	if err := TimedIntDBSSetUp("", true); err != nil {
		t.Fatal(err)
	}
	defer TimedIntDBSClose()

	if e := update([]byte{2, 0, 23, 44, 44, 56}, []byte{34}, *currentDB, true); e == nil {
		val, err := read([]byte{34}, 0, []int{2, 0}, *currentDB)
		fmt.Println(val, err)
	} else {
		fmt.Println(e)
	}
}

func Test_DBS(t *testing.T) {
	if err := TimedIntDBSSetUp("", true); err != nil {
		t.Fatal(err)
	}
	defer TimedIntDBSClose()

	if f, err := SetSeries("entry___notame__current_", 2, false); err != nil {
		fmt.Println(f)
		t.Fatal(err)
	} else {
		fmt.Println(f)
		if f {
			fmt.Println("Serie definition:", GetDefinition("entry___notame__current_"))
		}
	}

	if h, e := ReadHeader("entry___notame__current_", false); e != nil {
		t.Fatal(e)
	} else {
		fmt.Println("HEADER: ", h)
	}

	time.Sleep(2 * time.Second)
	ts := supp.Timestamp()
	ts = supp.Timestamp()
	a := SeriesSample{"entry___notame__current_", ts, 11}
	if err := StoreSampleTS(&a, false, true); err != nil {
		t.Fatal(err)
	}
	s0 := SeriesSample{"entry___notame__current_", ts - 500000, 8}
	s1 := SeriesSample{"entry___notame__current_", ts + 1000, 8}
	if tag, ts, vals, e := ReadSeriesTS(&s0, &s1, false); e != nil {
		t.Fatal(e)
	} else {
		fmt.Println(s1.UnmarshalSliceSS(tag, ts, vals))
	}

	//if tag, ts, vals, e := ReadLastNTS(&s1, 3, []int{}, false); e != nil {
	if tag, ts, vals, e := ReadLastNTS(&s1, 100, false); e != nil {
		t.Fatal(e)
	} else {
		fmt.Println(s1.UnmarshalSliceSS(tag, ts, vals))
	}

	if h, e := ReadHeader("entry___notame__current_", false); e != nil {
		t.Fatal(e)
	} else {
		fmt.Println("HEADER: ", h)
	}
}
