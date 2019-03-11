package support

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DevData struct {
	Tag  string // unique message Tag
	Ts   int64  // timestamp of last update
	Note string // possible description
	Data []int  // effective data
	Aggr bool   // true if needs to be cumulative
}

var DLog chan DevData

func setUpDevLogger() {
	DLog = make(chan DevData, 50)
	go devLogger(DLog)
}

func devLogger(data chan DevData) {

	r := func(d DevData, dt ...[]int) (msg string) {
		msg = d.Tag + ", " + strconv.Itoa(int(d.Ts)) + ", \"" + d.Note + "\""
		if len(dt) == 0 {
			for _, v := range d.Data {
				msg += ", " + strconv.Itoa(v)
			}
		} else {
			if len(dt[0]) != len(d.Data) {
				return ""
			}
			for i, v := range d.Data {
				msg += ", " + strconv.Itoa(v+dt[0][i])
			}
		}
		msg = strings.Trim(msg, " ")
		return
	}

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
		if d.Tag != "skip" {
			ct := time.Now().Local()
			pwd, _ := os.Getwd()
			_ = os.MkdirAll("log", os.ModePerm)
			file := filepath.Join(pwd, "log", ct.Format("2006-01-02")+".log")

			if input, err := ioutil.ReadFile(file); err != nil {
				if fn, err := os.Create(file); err != nil {
					log.Println("support.devLogger: error creating log: ", err)
				} else {
					//noinspection GoUnhandledErrorResult
					defer fn.Close()
					if _, err := fn.WriteString(r(d) + "\n"); err != nil {
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
						var nd []int
						for _, dt := range spv[3:] {
							if val, e := strconv.Atoi(strings.Trim(dt, " ")); e == nil {
								nd = append(nd, val)
							} else {
								log.Println("support.devLogger: error converting accruing data from log: ", e)
								nd = append(nd, 0)
							}
						}
						newc += r(d, nd) + "\n"
						adfile = false
					} else {
						if tmp := strings.Trim(v, " "); tmp != "" {
							newc += tmp + "\n"
						}

					}
				}
				if adfile {
					newc += r(d) + "\n"
				}
				if err = ioutil.WriteFile(file, []byte(newc), 0644); err != nil {
					log.Println("support.devLogger: error writing log: ", err)
				}
			}

		}
	}
}
