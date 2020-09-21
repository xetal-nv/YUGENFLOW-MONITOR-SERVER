package supp

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// development module for development logger

type DevData struct {
	Tag  string // unique message Tag
	Ts   int64  // timestamp of last update
	Note string // possible description
	Data []int  // effective data
	Aggr bool   // true if needs to be cumulative
}

var DLog chan DevData
var ODLog chan string

const bufDepth = 50

func setUpDevLogger() {
	DLog = make(chan DevData, bufDepth)
	ODLog = make(chan string, bufDepth)
	go devLogger(DLog, ODLog)
}

func devLogger(data chan DevData, out chan string) {

	r := func(d DevData, dt ...[]int) (msg string) {
		date := time.Unix(d.Ts/1000, 0).Format("Mon Jan:_2 15:04:0 2006")
		msg = d.Tag + ", " + date + ", \"" + d.Note + "\""
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
			go func() {
				DLog <- DevData{"supp.devLogger: recovering server",
					Timestamp(), "", []int{1}, true}
			}()
			log.Printf("supp.devLogger: recovering for crash\n ")
			go devLogger(data, out)
		}
	}()
	ct := time.Now().Local()
	pwd, _ := os.Getwd()
	_ = os.MkdirAll("log", os.ModePerm)
	file := filepath.Join(pwd, "log", "dvl_"+ct.Format("2006-01-02")+".log")
	for {
		d := <-data
		//ct := time.Now().Local()
		//pwd, _ := os.Getwd()
		//_ = os.MkdirAll("log", os.ModePerm)
		//file := filepath.Join(pwd, "log", ct.Format("2006-01-02")+".log")
		switch d.Tag {
		case "skip":
		case "read":
			var rt string
			if input, err := ioutil.ReadFile(file); err != nil {
				rt = "ERROR: File not fount"
			} else {
				rt = string(input)
			}
			go func() {
				select {
				case out <- rt:
				case <-time.After(2 * time.Second):
				}
			}()
		default:
			if input, err := ioutil.ReadFile(file); err != nil {
				if fn, err := os.Create(file); err != nil {
					log.Println("supp.devLogger: error creating log: ", err)
				} else {
					if _, err := fn.WriteString(r(d) + "\n"); err != nil {
						log.Println("supp.devLogger: error creating log: ", err)
					}
					//noinspection GoUnhandledErrorResult
					fn.Close()

				}
			} else {
				// read file and add or replace Tag
				newC := ""
				adFile := true
				for _, v := range strings.Split(strings.Trim(string(input), " "), "\n") {
					spv := strings.Split(v, ",")
					if strings.Trim(spv[0], " ") == d.Tag {
						var nd []int
						for _, dt := range spv[3:] {
							if val, e := strconv.Atoi(strings.Trim(dt, " ")); e == nil {
								nd = append(nd, val)
							} else {
								log.Println("supp.devLogger: error converting accruing data from log: ", e)
								nd = append(nd, 0)
							}
						}
						newC += r(d, nd) + "\n"
						adFile = false
					} else {
						if tmp := strings.Trim(v, " "); tmp != "" {
							newC += tmp + "\n"
						}

					}
				}
				if adFile {
					newC += r(d) + "\n"
				}
				if err = ioutil.WriteFile(file, []byte(newC), 0644); err != nil {
					log.Println("supp.devLogger: error writing log: ", err)
				}
			}

		}
	}
}
