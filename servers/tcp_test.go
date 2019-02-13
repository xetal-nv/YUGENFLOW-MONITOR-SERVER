package servers

import (
	"context"
	"countingserver/spaces"
	"fmt"
	"github.com/joho/godotenv"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
)

func Test_TCP_Setup(t *testing.T) {
	if e := godotenv.Load("../.env"); e != nil {
		t.Error("Error loading .env file")
	}
	spaces.SetUp()
	spaces.CountersSetpUp()
	SetUpTCP()

}

func Test_TCP_Connection(t *testing.T) {

	vals := []int{-2, -1, 0, 1, 2, 127}

	if e := godotenv.Load("../.env"); e != nil {
		t.Error("Error loading .env file")

	}

	spaces.SetUp()
	spaces.CountersSetpUp()
	SetUpTCP()

	go StartTCP(make(chan context.Context))

	for i := 0; i < 2; i++ {
		time.Sleep(2 * time.Second)
		fmt.Println(" TEST -> Connect to TCP channel")
		port := os.Getenv("TCPPORT")
		conn, e := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
		if e != nil {
			t.Error("Unable to connect")
		} else {
			//noinspection GoUnhandledErrorResult
			conn.Write([]byte{'a', 'b', 'c', 1, 2, 3})
			fmt.Println(" TEST -> Send illegal data")
			//noinspection GoUnhandledErrorResult
			conn.Write([]byte{'a'})
			for i := 0; i < 2; i++ {
				rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
				m := vals[rand.Intn(len(vals))]
				//noinspection GoUnhandledErrorResult
				conn.Write([]byte{1, 0, 2, byte(m)})
				time.Sleep(100 * time.Millisecond)

			}
		}
		//noinspection GoUnhandledErrorResult
		conn.Close()
		fmt.Println(" TEST -> Disconnect to TCP channel")
	}
}
