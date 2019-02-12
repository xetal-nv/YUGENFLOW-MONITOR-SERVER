package servers

import (
	"context"
	"countingserver/spaces"
	"fmt"
	"github.com/joho/godotenv"
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
	setUpTCP()

}

func Test_TCP_Connection(t *testing.T) {

	if e := godotenv.Load("../.env"); e != nil {
		t.Error("Error loading .env file")

	}

	go StartTCP(make(chan context.Context))

	time.Sleep(5 * time.Second)
	port := os.Getenv("TCPPORT")
	if conn, e := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port); e != nil {
		t.Error("Unable to connect")
	} else {
		conn.Write([]byte("c"))
		time.Sleep(2 * time.Second)
		fmt.Println("")
	}
}
