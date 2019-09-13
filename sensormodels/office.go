package sensormodels

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"gateserver/codings"
	"gateserver/support"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type pass struct {
	ts    int64
	entry int
	event int
}

type closure struct {
	ts  int64
	min int
	max int
}

type person struct {
	entries    [][]int    // entries groups the person can enter or exit
	off        []int      // gives number of left days of leave/sickness, how many times it was off/sick in a year, number of times it might still be off/sick this year
	deskHolder bool       // true is an deskHolder, false is a visitor (or similar)
	timing     [][]string // provide timing as [arrival, morning_pause, lunch, afternoon_pause, exit]
	pauses     []int      // provides [pauses_taken max_number_pauses] for the day
	fridayExit []string   // timing for friday ending, if "" it is set as normal time
}

// installation information
var gates = [][]int{{1, 2}, {3, 4}, {5, 6}, {7, 8}}
var macSensors = [][]byte{
	{'a', 'b', 'c', '1', '2', '1'},
	{'a', 'b', 'c', '1', '2', '2'},
	{'a', 'b', 'c', '1', '2', '3'},
	{'a', 'b', 'c', '1', '2', '4'},
	{'a', 'b', 'c', '1', '2', '5'},
	{'a', 'b', 'c', '1', '2', '6'},
	{'a', 'b', 'c', '1', '2', '7'},
	{'a', 'b', 'c', '1', '2', '8'},
}
var allEntries = [][]int{
	{2, 1},
	{4, 3},
}
var floor1Entries = [][]int{
	{2, 1},
}
var floor2Entries = [][]int{
	{4, 3},
}

var startDayTime string = "00:01"

// define a new person, timings and pauses provide the reference and they get randomized by the constructor
func newPerson(entries [][]int, timings []string, longmod []int, pauses int, deskHolder bool, friday string) (p person) {

	f := func(tm string, delta int) (t []string) {
		var hour, mins int
		var err error
		if tm != "" {
			tmp := strings.Split(tm, ":")
			hh, mm := tmp[0], tmp[1]

			if hour, err = strconv.Atoi(hh); err == nil {
				if mins, err = strconv.Atoi(mm); err == nil {
					if mins != 0 {
						m1 := "0" + strconv.Itoa((mins-delta)+delta/mins*60)
						//m2 := "0" + strconv.Itoa((mins+delta)%60)
						t = []string{
							strconv.Itoa(hour-delta/mins) + ":" + m1[len(m1)-2:],
							//strconv.Itoa(hour+(mins+delta)/60) + ":" + m2[len(m2)-2:]}
							strconv.Itoa(2 * delta)}
					} else {
						m1 := "0" + strconv.Itoa(60-delta)
						//m2 := "0" + strconv.Itoa((mins+delta)%60)
						t = []string{
							strconv.Itoa(hour-1) + ":" + m1[len(m1)-2:],
							//strconv.Itoa(hour+(mins+delta)/60) + ":" + m2[len(m2)-2:]}
							strconv.Itoa(2 * delta)}
					}
				}
			}
			if err != nil {
				log.Fatalf("newPerson failed to convert %v into an integer\n", tmp)
			}
		} else {
			t = []string{}
		}
		return
	}

	p.entries = entries
	p.deskHolder = deskHolder
	p.off = []int{0, 0, support.RandInt(0, 8)}
	p.pauses = []int{0, 1 + pauses + support.RandInt(0, 3)}
	for i, v := range timings {
		p.timing = append(p.timing, f(v, support.RandInt(1, longmod[i])))
	}
	if friday == "" {
		p.fridayExit = p.timing[len(p.timing)-1]
	} else {
		p.fridayExit = []string{friday, strconv.Itoa(longmod[len(longmod)-1])}
	}

	return p
}

// activate a person, return the true is the person enter (false is exits) and the time it will do so
func (p person) activate() (valid bool, ev []pass) {
	//fmt.Println(p)
	valid = false
	if p.off[0] == 0 {
		//  determine if the person is sick
		if p.off[1] < p.off[2] {
			if support.RandInt(0, 10) == 5 {
				// person is sick
				p.off[1] += 1
				p.off[0] = support.RandInt(1, 30)
				return
			}
		}

		valid = true
		t := time.Now()
		fl := -1
		midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Unix()
		npMorning := support.RandInt(1, p.pauses[1])
		npAfternoon := 0
		if npMorning+1 < p.pauses[1] {
			npAfternoon = support.RandInt(0, p.pauses[1]-npMorning-1)
		}
		p.pauses[0] = npMorning + npAfternoon + 1
		for i, tm := range p.timing {
			f := func(v []string, event int, oride closure, lastvalid int64) (bool, pass) {
				if val, err := strconv.Atoi(tm[1]); err == nil {
					var del int
					if oride.ts == 0 {
						del = support.RandInt(0, val)
					} else {
						del = support.RandInt(oride.min, oride.max)
					}
					if hour, err := strconv.Atoi(v[0]); err == nil {
						if mins, err := strconv.Atoi(v[1]); err == nil {
							var ts int64
							if oride.ts == 0 {
								ts = midnight + int64(hour*3600) + int64(mins*60)
								if ts < lastvalid {
									ts = lastvalid
								}
							} else {
								if oride.ts < lastvalid {
									ts = lastvalid
								} else {
									ts = oride.ts
								}
							}
							ts += int64((del/60)*3600) + int64((del%60)*60)
							var ent int
							if fl == -1 {
								fl = support.RandInt(0, len(p.entries)-1)
								ent = p.entries[fl][support.RandInt(0, len(p.entries[fl])-1)]
							} else {
								ent = p.entries[fl][support.RandInt(0, len(p.entries[fl])-1)]
								fl = -1
							}
							return true, pass{ts, ent, event}
						}
					}
				}
				return false, pass{}
			}
			if i == 0 || i == 4 {
				//fmt.Println(tm)
				v := strings.Split(tm[0], ":")
				// entry/exit
				event := 1
				lv := int64(0)
				if i == 4 {
					event = -1
					//noinspection GoNilness
					lv = ev[len(ev)-1].ts
					// check if friday (5) and set the start at 15:45
					if today := int(time.Now().Weekday()); today == 5 {
						v = strings.Split(p.fridayExit[0], ":")
					}
				}

				if ok, vi := f(v, event, closure{ts: 0}, lv); ok {
					ev = append(ev, vi)
				}
			} else if p.deskHolder {
				v := strings.Split(tm[0], ":")
				switch i {
				case 1:
					//if p.deskHolder {
					// morning
					//var ok bool
					vin := pass{ts: 0}
					//vout := pass{ts: 0}
					for j := 0; j <= npMorning; j++ {
						if ok, vout := f(v, -1, closure{ts: vin.ts, min: 10, max: 15}, vin.ts); ok {
							if ok, vin = f(v, 1, closure{vout.ts, 5, 15}, vout.ts); ok {
								ev = append(ev, vout)
								ev = append(ev, vin)
							}
						}
					}
					//} else {
					//	// visitor
					//	if ok, v := f(v, -1, closure{ts: 0}); ok {
					//		ev = append(ev, v)
					//	}
					//}
				case 2:
					//lunch
					//noinspection GoNilness
					if ok, vout := f(v, -1, closure{ts: 0}, ev[len(ev)-1].ts); ok {
						if ok, vin := f(v, 1, closure{vout.ts, 15, 40}, vout.ts); ok {
							ev = append(ev, vout)
							ev = append(ev, vin)
						}
					}
				case 3:
					// afternoon
					//var ok bool
					vin := pass{ts: 0}
					//vout := pass{ts: 0}
					// check if friday (5), if person is set to leave early, remove afteroon pauses
					if today := int(time.Now().Weekday()); today == 5 && (p.fridayExit[0] != p.timing[len(p.timing)-1][0]) {
						npAfternoon = 0
					}
					for j := 0; j <= npAfternoon; j++ {
						//noinspection GoNilness
						if ok, vout := f(v, -1, closure{ts: vin.ts, min: 10, max: 15}, ev[len(ev)-1].ts); ok {
							if ok, vin := f(v, 1, closure{vout.ts, 5, 15}, vout.ts); ok {
								ev = append(ev, vout)
								ev = append(ev, vin)
							}
						}
					}
				default:
					// never happens
				}
			}
		}
	} else {
		p.off[0] -= 1
	}
	return
}

// reduce sick days by one
func (p person) sickDay() {
	if p.off[0] > 0 {
		p.off[0] -= 1
	}
}

var count = 0
var mutex = &sync.Mutex{}

// gate implements a gate and need to serve two sensors on two TCP channels in the proper manner
func gate(in chan int, sensors []int) {
	port := os.Getenv("TCPPORT")
	connS1, e1 := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	connS2, e2 := net.Dial(os.Getenv("TCPPROT"), "0.0.0.0:"+port)
	if e1 != nil || connS1 == nil || e2 != nil || connS2 == nil {
		fmt.Println("Unable to connect for sensors", sensors, e1, e2)
		os.Exit(1)
		// send associated macSensors addresses

	} else {
		//noinspection GoUnhandledErrorResult
		defer connS1.Close()
		//noinspection GoUnhandledErrorResult
		defer connS2.Close()
		//noinspection GoUnhandledErrorResult
		connS1.Write(macSensors[sensors[0]-1])
		//noinspection GoUnhandledErrorResult
		connS2.Write(macSensors[sensors[1]-1])

		// set-up for receiving data and/or commands from the server
		c1 := make(chan []byte)
		c2 := make(chan []byte)
		r := func(conn net.Conn, c chan []byte) {
			var e error
			for e == nil {
				cmd := make([]byte, 1)
				ll := 1
				if _, e = conn.Read(cmd); e == nil {
					if l, ok := cmdargs[cmd[0]]; ok {
						ll += l
					}
					cmde := make([]byte, ll)
					if _, e = conn.Read(cmde); e == nil {
						cmd = append(cmd, cmde...)
					}
					if e == nil {
						select {
						case c <- cmd:
						case <-time.After(40 * time.Second):
						}
					}
				}
			}
		}

		go r(connS1, c1)
		go r(connS2, c2)

		// loop to send new data or handle received data
		f := func(conn net.Conn, v []byte) {
			crc := codings.Crc8(v[:len(v)-1])
			if crc == v[len(v)-1] {
				msg := []byte{v[0]}
				if rt, ok := command[v[0]]; ok {
					msg = append(msg, rt...)
				}
				crc = codings.Crc8(msg)
				msg = append(msg, crc)
				//noinspection GoUnhandledErrorResult
				conn.Write(msg)
				if v[0] == 14 {
					time.Sleep(10 * time.Second)
				}
			}
		}

		fmt.Printf("OFFICE TEST: Started entry with gates %v and %v\n", sensors[0], sensors[1])

		for {
			select {
			case data := <-in:

				mutex.Lock()
				count += data
				fmt.Println("count is", count)
				mutex.Unlock()

				bs1 := make([]byte, 4)
				binary.BigEndian.PutUint32(bs1, uint32(sensors[0]))
				//noinspection GoUnhandledErrorResult
				msg1 := []byte{1, bs1[2], bs1[3], byte(data)}
				msg1 = append(msg1, codings.Crc8(msg1))
				bs2 := make([]byte, 4)
				binary.BigEndian.PutUint32(bs2, uint32(sensors[1]))
				//noinspection GoUnhandledErrorResult
				msg2 := []byte{1, bs2[2], bs2[3], byte(data)}
				msg2 = append(msg2, codings.Crc8(msg2))
				//noinspection GoUnhandledErrorResult
				connS1.Write(msg1)
				time.Sleep(300 * time.Millisecond)
				//noinspection GoUnhandledErrorResult
				connS2.Write(msg2)
				if support.Debug != 0 {
					if data == 1 {
						fmt.Printf("OFFICE TEST: Sending to %v:%v and %v:%v -> 1\n", sensors[0], macSensors[sensors[0]-1], sensors[1], macSensors[sensors[1]-1])
					} else {
						fmt.Printf("OFFICE TEST: Sending to %v:%v and %v:%v -> -1\n", sensors[0], macSensors[sensors[1]-1], sensors[1], macSensors[sensors[0]-1])
					}
				}
			case v := <-c1:
				f(connS1, v)
			case v := <-c2:
				f(connS2, v)
			}
		}
	}
}

// This module can be used to faithfully emulate real offices
func Office() {
	support.RandomInit()

	//define gates
	var out []chan int
	for i := 0; i < 4; i++ {
		tmp := make(chan int, 10)
		out = append(out, tmp)
		go gate(tmp, gates[i])
	}

	// define the people using the offices\
	var allPeople []person

	// Employees
	for i := 0; i < 10; i++ {
		// employees floor1
		allPeople = append(allPeople, newPerson(floor1Entries, []string{"9:00", "10:00", "12:30", "15:15", "17:30"}, []int{30, 30, 15, 15, 45},
			2, true, "15:00"))
		// employees floor2
		allPeople = append(allPeople, newPerson(floor2Entries, []string{"9:00", "10:00", "12:30", "15:15", "17:30"}, []int{30, 30, 15, 15, 45},
			2, true, "15:00"))
	}
	for i := 0; i < 5; i++ {
		// employees both floors
		allPeople = append(allPeople, newPerson(allEntries, []string{"9:00", "10:00", "12:30", "15:15", "17:30"}, []int{30, 30, 15, 15, 45},
			2, true, ""))
	}

	// Visitors
	for i := 0; i < support.RandInt(0, 10); i++ {
		allPeople = append(allPeople, newPerson(allEntries, []string{"9:00", "", "", "", "16:00"}, []int{30, 30, 15, 15, 45},
			0, false, ""))
	}
	//guard
	allPeople = append(allPeople, newPerson(allEntries, []string{"19:45", "", "", "", "22:00"}, []int{30, 30, 15, 15, 30},
		0, false, ""))

	//// for debug
	//tmpH := time.Now().Hour()
	//tmpM := time.Now().Minute() + 10
	//if tmpM > 60 {
	//	tmpH++
	//	if tmpH >= 24 {
	//		fmt.Println("crossing day")
	//		os.Exit(1)
	//	}
	//	tmpM %= 60
	//}
	//tmp1 := strconv.Itoa(tmpH) + ":" + strconv.Itoa(tmpM)
	//tmp2 := strconv.Itoa(tmpH) + ":" + strconv.Itoa(tmpM+10)
	//allPeople = append(allPeople, newPerson(allEntries, []string{tmp1, "", "", "", tmp2}, []int{5, 30, 15, 15, 5},
	//	0, false))

	// Load holiday information
	filename := "./sensormodels/holidays_BE_" + strconv.Itoa(time.Now().Year()) + ".json"

	var holidays Response
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		//fmt.Println(err)
		fmt.Println("OFFICE TEST: Access API for new Holidays list")
		holidays = extractHolidays("BE", "2019")
		file, _ := json.MarshalIndent(holidays, "", " ")
		_ = ioutil.WriteFile(filename, file, 0644)
	} else {
		fmt.Println("OFFICE TEST: Read Holidays file:", filename)
		_ = json.Unmarshal([]byte(file), &holidays)
	}

	// starts the daily cycle
	for {

		today := strconv.Itoa(time.Now().Year()) + "-" + strconv.Itoa(int(time.Now().Month())) + "-" + strconv.Itoa(time.Now().Day())
		//  check is day is holidays
		isHolidays := false
		for _, v := range holidays.Holidays.Holidays {
			if v.Public {
				isHolidays = v.Date == today
			}
		}

		// generate the current day events

		// We exclude the week end
		if today := time.Now().Weekday(); today != 6 && today != 7 && !isHolidays {
			fmt.Println("OFFICE TEST: Starting a new working day:", time.Now().Weekday(), time.Now().Day(), ",", time.Now().Month(), ",", time.Now().Year())
			var dayEvents []pass

			for j, v := range allPeople {
				//for _, v := range allPeople {
				if valid, data := v.activate(); valid {

					// for debug
					for i := 1; i < len(data); i++ {
						if data[i].ts < data[i-1].ts {
							fmt.Println("error", j, i, data)
						}
					}
					//count := 0
					//for _, k := range data {
					//	count += k.event
					//	fmt.Print(count, " ")
					//}
					//fmt.Println()

					dayEvents = append(dayEvents, data...)
				}
			}

			sort.SliceStable(dayEvents, func(p, q int) bool {
				return dayEvents[p].ts < dayEvents[q].ts
			})

			//for debug
			//os.Exit(1)
			//for _, v := range dayEvents {
			//	fmt.Println("OFFICE TEST: ", time.Unix(v.ts, 0), v)
			//}
			//fmt.Println("")
			//os.Exit(1)

			timeNow := time.Now().Unix()

			// remove all older samples (useful for the first run only)
			i := 0
			loop := true
			for loop {
				if i >= len(dayEvents) {
					loop = false
				} else if timeNow <= dayEvents[i].ts-60 {
					loop = false
				} else {
					i++
				}
			}

			for i < len(dayEvents) {
				//fmt.Println("serving", dayEvents[i], "with index", i)
				for timeNow >= dayEvents[i].ts-60 {
					// for debug
					//fmt.Println(time.Unix(timeNow, 0), "-> ", time.Unix(dayEvents[i].ts, 0))
					//if dayEvents[i].entry-1 > len(out) {
					//	fmt.Println("OFFICE TEST: fatal error, out of index", dayEvents[i])
					//	os.Exit(1)
					//}
					//fmt.Println("event served is", dayEvents[i])

					//noinspection GoNilness
					out[dayEvents[i].entry-1] <- dayEvents[i].event
					if i += 1; i >= len(dayEvents) {
						break
					}
				}
				time.Sleep(40 * time.Second)
				timeNow = time.Now().Unix()
			}
		} else {
			if isHolidays {
				fmt.Println("OFFICE TEST: Today is Holidays")
			} else {
				fmt.Println("OFFICE TEST: Today is WE")
			}
		}

		// wait for the next day
		if ns, err := time.Parse(support.TimeLayout, startDayTime); err != nil {
			fmt.Println("OFFICE TEST: Syntax error in specified start time", startDayTime)
			os.Exit(1)
		} else {
			now := time.Now()
			nows := strconv.Itoa(now.Hour()) + ":"
			mins := "00" + strconv.Itoa(now.Minute())
			nows += mins[len(mins)-2:]
			if ne, err := time.Parse(support.TimeLayout, nows); err != nil {
				log.Println("Syntax error retrieving system current time")
				os.Exit(1)
			} else {
				if ns.Hour() >= ne.Hour() {
					del := ns.Sub(ne)
					//log.Println("Waiting till", startDayTime, "before starting server")
					time.Sleep(del)
				} else {
					del := 24*60 - ne.Hour()*60 - ne.Minute() + ns.Hour()*60 + ns.Minute()
					//del := ne.Sub(ns)
					//del := 24 - ne.Sub(ns)
					//log.Println("Waiting till", del, "before starting server")
					time.Sleep(time.Duration(del) * time.Minute)
				}
			}
		}
	}
}
