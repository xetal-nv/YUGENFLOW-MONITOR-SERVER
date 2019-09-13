package sensormodels

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net"
	. "os"
	"strconv"
	"testing"
	"time"
)

func Test_office(t *testing.T) {

	if e := godotenv.Load("../.envtest_office"); e != nil {
		panic("Fatal error:" + e.Error())
	}
	if e := godotenv.Load("../.systemenv"); e != nil {
		panic("Fatal error:" + e.Error())
	}

	h := func(conn net.Conn) {
		for {
			buffer := make([]byte, 6)
			//noinspection GoUnhandledErrorResult
			conn.Read(buffer)
			fmt.Println("Server received:", buffer)
		}
	}

	s := func() {
		port := Getenv("TCPPORT")
		l, e := net.Listen(Getenv("TCPPROT"), "0.0.0.0:"+port)
		if e != nil {
			log.Fatal("servers.StartTCP: fatal error:", e)
		}
		//noinspection GoUnhandledErrorResult
		defer l.Close()
		log.Printf("servers.StartTCP: listening on 0.0.0.0:%v\n", port)
		for {
			conn, e := l.Accept()
			if e != nil {
				log.Printf("servers.StartTCP: Error accepting: %v\n", e)
				if l != nil {
					_ = l.Close()
				}
			}
			// Handle connections in a new goroutine.
			go h(conn)
		}
	}

	go s()

	Office()
}

func Test_HAPI(t *testing.T) {

	filename := "holidays_BE_2019.json"

	var holidays Response
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Access API")
		holidays = extractHolidays("BE", "2019")
		file, _ := json.MarshalIndent(holidays, "", " ")
		_ = ioutil.WriteFile(filename, file, 0644)
	} else {
		fmt.Println("Read file")
		_ = json.Unmarshal([]byte(file), &holidays)
	}

	today := strconv.Itoa(time.Now().Year()) + "-" + strconv.Itoa(int(time.Now().Month())) + "-" + strconv.Itoa(time.Now().Day())
	for _, v := range holidays.Holidays.Holidays {
		if v.Public {
			if v.Date == today {
				fmt.Println("Today is an holiday")
			}
		}
	}

}
