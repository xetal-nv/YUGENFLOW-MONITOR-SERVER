package servers

import (
	"context"
	"countingserver/gates"
	"countingserver/spaces"
	"countingserver/storage"
	"countingserver/support"
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func Test_Registers(t *testing.T) {
	a := make(chan interface{})
	b := make(chan interface{})
	go storage.SafeReg(a, b)
	a <- 2
	if <-b != 2 {
		t.Fatalf("Expected %v but got %v", -1, <-b)
	}
}

func Test_SETUP(t *testing.T) {
	support.SupportSetUp("../.env")
	if err := storage.TimedIntDBSSetUp(false); err != nil {
		t.Fatal(err)
	}
	defer storage.TimedIntDBSClose()
	gates.SetUp()
	spaces.SetUp()
	time.Sleep(5 * time.Second)
}

func Test_TCP_StreamDBS(t *testing.T) {

	iter := 20

	support.SupportSetUp("../.env")

	if err := storage.TimedIntDBSSetUp(false); err != nil {
		t.Fatal(err)
	}
	defer storage.TimedIntDBSClose()

	vals := []int{-1, 0, 1, 2, 127}
	counter := 0

	var avgws []string
	avgws = append(avgws, "current")
	if avgw := strings.Trim(os.Getenv("SAVEWINDOW"), ";"); avgw == "" {
		t.Fatalf("Error in .env file, SAVEWINDOW is empty")
	} else {
		for _, v := range strings.Split(avgw, ";") {
			name := strings.Trim(strings.Split(strings.Trim(v, " "), " ")[0], " ")
			avgws = append(avgws, name)
		}
	}
	neg := os.Getenv("INSTANTNEG")
	gates.SetUp()
	spaces.SetUp()
	go StartTCP(make(chan context.Context))

	r := func() (ret bool) {
		ret = true
		time.Sleep(2 * time.Second)
		fmt.Println(" TEST -> Connect to TCP channel")
		port := os.Getenv("TCPPORT")
		conn, e := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
		if e != nil {
			t.Fatalf("Unable to connect")
		} else {
			//noinspection GoUnhandledErrorResult
			conn.Write([]byte{'a', 'b', 'c', 1, 2, 3})
			for i := 0; i < iter; i++ {
				rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
				m := vals[rand.Intn(len(vals))]
				if m != 127 {
					counter += m
					if counter < 0 && neg == "0" {
						counter = 0
					}
				}
				//noinspection GoUnhandledErrorResult
				conn.Write([]byte{1, 0, 1, byte(m)})
				time.Sleep(1000 * time.Millisecond)

			}
		}
		//noinspection GoUnhandledErrorResult
		conn.Close()
		fmt.Println(" TEST -> Disconnect to TCP channel")
		time.Sleep(30 * time.Second)
		a := new(storage.SerieSample)
		if e := a.Extract(<-spaces.LatestBankOut[support.StringLimit("sample", support.LabelLength)][support.StringLimit("noname",
			support.LabelLength)][support.StringLimit("current", support.LabelLength)]); e != nil {
			t.Fatalf("Invalid value from the current register")
		}
		if a.Val() != counter {
			fmt.Println("TEST Failed (e/r):", counter, a.Val())
			//t.Fatalf("Expected counter is not as real counter")
			ret = false
		}

		fmt.Println("Expected result is", counter)
		var wg sync.WaitGroup
		end := make(chan (bool))
		for _, v := range avgws {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				a := reflect.ValueOf(<-spaces.LatestBankOut[support.StringLimit("sample",
					support.LabelLength)][support.StringLimit("noname", support.LabelLength)][support.StringLimit(name, support.LabelLength)])
				fmt.Println("Check sample", name, "pipe ::", a)
				b := reflect.ValueOf(<-spaces.LatestBankOut[support.StringLimit("entry",
					support.LabelLength)][support.StringLimit("noname", support.LabelLength)][support.StringLimit(name, support.LabelLength)])
				fmt.Println("Check entry", name, "pipe ::", b)
			}(v)
		}
		go func() {
			wg.Wait()
			end <- true
		}()
		select {
		case <-end:
		case <-time.After(5 * time.Second):
			//t.Fatal("Hanging on register read")
			fmt.Println("Hanging on register read")
			ret = false
		}
		return
	}

	ok := 0
	tot := 10
	for i := 0; i < tot; i++ {
		if r() {
			ok += 1
		}
		time.Sleep(2 * time.Second)
	}
	fmt.Printf("Success %v on %v\n", ok, tot)
	if ok != tot {
		t.Fatal("Failed")
	}
}
