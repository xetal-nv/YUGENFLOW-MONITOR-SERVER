package registers

import (
	"countingserver/support"
	"fmt"
	"testing"
	"time"
)

func Test_Setup(t *testing.T) {
	TimedIntDBSSetUp()
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
	TimedIntDBSSetUp()
	defer TimedIntDBSClose()

	if f, err := SetSeries("test", 30); err != nil {
		t.Fatal(err)
	} else {
		fmt.Println("Found:", f)
	}
	time.Sleep(2 * time.Second)
	if err := StoreSerieSample("test", support.Timestamp(), 3); err != nil {
		t.Fatal(err)
	}

	//if c, e := read([]byte{0}, 28, *currentDB); e != nil {
	//	t.Fatal(e)
	//} else {
	//	fmt.Println(c.unmarshall())
	//}
}
