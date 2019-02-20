package servers

import (
	"context"
	"countingserver/gates"
	"countingserver/registers"
	"countingserver/spaces"
	"countingserver/support"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func Test_Registers(t *testing.T) {
	a := make(chan int)
	b := make(chan int)
	go registers.IntCell("", a, b)
	if <-b != -1 {
		t.Fatalf("Expected %v but got %v", -1, <-b)
	}
}

func Test_SETUP(t *testing.T) {
	support.SupportSetUp("../.env")
	if err := registers.TimedIntDBSSetUp(); err != nil {
		t.Fatal(err)
	}
	defer registers.TimedIntDBSClose()
	gates.SetUp()
	spaces.SetUp()
}

func TCP_Connection(vals []int) string {
	counter := 0
	support.SupportSetUp("../.env")
	neg := os.Getenv("INSTANTNEG")
	gates.SetUp()
	spaces.SetUp()

	go StartTCP(make(chan context.Context))

	for i := 0; i < 3; i++ {
		time.Sleep(2 * time.Second)
		fmt.Println(" TEST -> Connect to TCP channel")
		port := os.Getenv("TCPPORT")
		conn, e := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
		if e != nil {
			return "Unable to connect"
		} else {
			//noinspection GoUnhandledErrorResult
			conn.Write([]byte{'a', 'b', 'c', 1, 2, 3})
			fmt.Println(" TEST -> Send other data")
			//noinspection GoUnhandledErrorResult
			conn.Write([]byte{7, 1, 1})
			for i := 0; i < 10; i++ {
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
		fmt.Println(" TEST -> Send illegal data")
		//noinspection GoUnhandledErrorResult
		conn.Write([]byte{37, 1, 1})
		//noinspection GoUnhandledErrorResult
		conn.Close()
		fmt.Println(" TEST -> Disconnect to TCP channel")
	}
	time.Sleep(5 * time.Second)
	a := <-spaces.LatestDataBankOut["noname"]["current"]

	if a.Ct != counter {
		fmt.Println("TEST Failed:", counter, a.Ct)
		return "Expected counter is not as real counter"
	}

	spaces.ResetDataDBS["noname"]["current"] <- false

	return ""
}

func Test_TCP_ConnectionNeg(t *testing.T) {

	if res := TCP_Connection([]int{-2, -1, 0, 127}); res != "" {
		t.Fatalf("Test Neg: " + res + "\n")
	}
}

func Test_TCP_ConnectionAll(t *testing.T) {
	if res := TCP_Connection([]int{-2, -1, 0, 1, 2, 127}); res != "" {
		t.Fatalf("Test Full: " + res + "\n")
	}
}

func Test_TCP_StreamDBS(t *testing.T) {

	support.SupportSetUp("../.env")

	if err := registers.TimedIntDBSSetUp(); err != nil {
		t.Fatal(err)
	}
	defer registers.TimedIntDBSClose()

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

	time.Sleep(2 * time.Second)
	fmt.Println(" TEST -> Connect to TCP channel")
	port := os.Getenv("TCPPORT")
	conn, e := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	if e != nil {
		t.Fatalf("Unable to connect")
	} else {
		//noinspection GoUnhandledErrorResult
		conn.Write([]byte{'a', 'b', 'c', 1, 2, 3})
		for i := 0; i < 100; i++ {
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
	a := <-spaces.LatestDataBankOut["noname"]["current"]

	if a.Ct != counter {
		fmt.Println("TEST Failed:", counter, a.Ct)
		t.Fatalf("Expected counter is not as real counter")
	}

	for _, v := range avgws {
		go func(name string) { fmt.Println("Check", name, "pipe ::", <-spaces.LatestDataBankOut["noname"][name]) }(v)
	}
}
