package sensormodels

import (
	"fmt"
	"gateserver/support"
	"log"
	"sort"
	"strconv"
	"strings"
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
	entries  [][]int    // entries groups the person can enter or exit
	sickness []int      // gives number of left days of sickness, how many times it was sick in a year, number of times it might still get sick this year
	employee bool       // true is an employee, false is a visitor (or similar)
	timing   [][]string // provide values for arrival, pause, lunch, pause and departure as [min max]
	pauses   []int      // provides [pauses_taken max_number_pauses] for the day
}

// define a new person, timings and pauses provide the reference and they get randomized by the constructor
func newPerson(entries [][]int, timings []string, longmod []int, pauses int, employee bool) (p person) {

	f := func(tm string, delta int) (t []string) {
		var hour, mins int
		var err error
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
		return
	}

	p.entries = entries
	p.employee = employee
	p.sickness = []int{0, support.RandInt(0, 5)}
	p.pauses = []int{0, 1 + pauses + support.RandInt(0, 3)}
	for i, v := range timings {
		p.timing = append(p.timing, f(v, support.RandInt(1, longmod[i])))
	}

	return p
}

// activate a person, return the true is the person enter (false is exits) and the time it will do so
func (p person) activate() (ev []pass) {
	t := time.Now()
	fl := -1
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Unix()
	npMorning := support.RandInt(1, p.pauses[1])
	npAfteroon := 0
	if npMorning+1 < p.pauses[1] {
		npAfteroon = support.RandInt(0, p.pauses[1]-npMorning-1)
	}
	p.pauses[0] = npMorning + npAfteroon + 1
	for i, tm := range p.timing {
		v := strings.Split(tm[0], ":")
		f := func(event int, oride closure) (bool, pass) {
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
							ts = midnight + int64((hour+del/60)*3600) + int64((mins+del%60)*60)
						} else {
							ts = oride.ts + int64((del/60)*3600) + int64((del%60)*60)
						}
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
			// entry/exit
			event := 1
			if i == 4 {
				event = -1
			}
			if ok, v := f(event, closure{ts: 0}); ok {
				ev = append(ev, v)
			}
		} else {
			switch i {
			case 1:
				if p.employee {
					// morning
					for j := 0; j <= npMorning; j++ {
						if ok, vout := f(-1, closure{ts: 0}); ok {
							if ok, vin := f(1, closure{vout.ts, 5, 15}); ok {
								ev = append(ev, vout)
								ev = append(ev, vin)
							}
						}
					}
				} else {
					// visitor
					if ok, v := f(-1, closure{ts: 0}); ok {
						ev = append(ev, v)
					}
				}
			case 2:
				//lunch
				if ok, vout := f(-1, closure{ts: 0}); ok {
					if ok, vin := f(1, closure{vout.ts, 15, 40}); ok {
						ev = append(ev, vout)
						ev = append(ev, vin)
					}
				}
			case 3:
				// afternoon
				for j := 0; j <= npAfteroon; j++ {
					if ok, vout := f(-1, closure{ts: 0}); ok {
						if ok, vin := f(1, closure{vout.ts, 5, 15}); ok {
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
	return
}

// This module can be used to faithfully emulate real offices
func Office() {
	support.RandomInit()

	// define the people using the offices
	var allPeople []person
	allEntries := [][]int{
		{0, 1},
		{2, 3},
	}
	floor1Entries := [][]int{
		{0, 1},
	}
	floor2Entries := [][]int{
		{2, 3},
	}

	// Employees
	for i := 0; i < 10; i++ {
		// employees floor1
		allPeople = append(allPeople, newPerson(floor1Entries, []string{"8:30", "10:00", "12:30", "15:15", "17:30"}, []int{30, 30, 15, 15, 45},
			2, true))
		// employees floor2
		allPeople = append(allPeople, newPerson(floor2Entries, []string{"8:30", "10:00", "12:30", "15:15", "17:30"}, []int{30, 30, 15, 15, 45},
			2, true))
	}
	for i := 0; i < 5; i++ {
		// employees both floors
		allPeople = append(allPeople, newPerson(allEntries, []string{"8:30", "10:00", "12:30", "15:15", "17:30"}, []int{30, 30, 15, 15, 45},
			2, true))
	}

	// Visitors
	for i := 0; i < support.RandInt(0, 5); i++ {
		allPeople = append(allPeople, newPerson(allEntries, []string{"9:00", "16:00"}, []int{55, 55},
			0, false))
	}

	// generate the current day events
	var dayEvents []pass
	for _, v := range allPeople {
		dayEvents = append(dayEvents, v.activate()...)
	}

	sort.SliceStable(dayEvents, func(p, q int) bool {
		return dayEvents[p].ts < dayEvents[q].ts
	})

	k := 0
	for _, v := range dayEvents {
		fmt.Println(time.Unix(v.ts, 0), ":", v.entry, ":", v.event)
		k += v.event
	}
	fmt.Println(k)

}
