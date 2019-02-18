package support

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type DevData struct {
	Tag  string // unique message Tag
	Ts   int64  // timestamp of last update
	Note string // possible description
	Data []int  // effective data
}

var DLog chan DevData

func SetUpDevLogger() {
	DLog = make(chan DevData, 30)
	go devLogger(DLog)
}

func devLogger(data chan DevData) {
	defer func() {
		if e := recover(); e != nil {
			if e != nil {
				log.Printf("support.devLogger: recovering for crash\n ")
				go devLogger(data)
			}
		}
	}()
	for {
		d := <-data
		msg := d.Tag + ", " + strconv.Itoa(int(d.Ts)) + ", \"" + d.Note + "\""
		for _, v := range d.Data {
			msg += ", " + strconv.Itoa(v)
		}
		msg = strings.Trim(msg, " ")
		if d.Tag != "skip" {
			ct := time.Now().Local()
			file := ct.Format("2006-01-02") + ".log"

			if input, err := ioutil.ReadFile(file); err != nil {
				if fn, err := os.Create(file); err != nil {
					log.Println("support.devLogger: error creating log: ", err)
				} else {
					//noinspection GoUnhandledErrorResult
					defer fn.Close()
					if _, err := fn.WriteString(msg + "\n"); err != nil {
						log.Println("support.devLogger: error creating log: ", err)
					}
				}
			} else {
				// read file and add or replace Tag
				newc := ""
				adfile := true
				for _, v := range strings.Split(strings.Trim(string(input), " "), "\n") {
					spv := strings.Split(v, ",")
					if strings.Trim(spv[0], " ") == d.Tag {
						newc += msg + "\n"
						adfile = false
					} else {
						if tmp := strings.Trim(v, " "); tmp != "" {
							newc += tmp + "\n"
						}

					}
				}
				if adfile {
					newc += msg + "\n"
				}
				if err = ioutil.WriteFile(file, []byte(newc), 0644); err != nil {
					log.Println("support.devLogger: error writing log: ", err)
				}
			}

		}
	}
}
