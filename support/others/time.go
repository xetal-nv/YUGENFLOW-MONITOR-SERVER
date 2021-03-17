package others

import (
	"gateserver/support/globals"
	"math/rand"
	"strconv"
	"time"
)

var seed = time.Now().UnixNano()

func WaitRandom(reference int) {
	wait := rand.New(rand.NewSource(seed)).Intn(reference)
	time.Sleep(time.Duration(wait) * time.Second)
}

// used by InClosureTime, InClosureTimeFull
func inTimeSpan(start, end, check time.Time) bool {
	if check.After(end) {
		if end.After(start) {
			return false
		} else {
			return check.After(start)
		}
	}
	if end.After(start) {
		return check.After(start)
	}
	return check.Before(start)
}

// true if current time in between start and end
func InClosureTime(start, end time.Time) (rt bool, err error) {
	if start == end {
		return false, nil
	}
	// this ensures the first minute if captured
	start = start.Add(-1 * time.Second)
	now := time.Now()
	nows := strconv.Itoa(now.Hour()) + ":"
	mins := "00" + strconv.Itoa(now.Minute())
	nows += mins[len(mins)-2:]
	var ns time.Time
	ns, err = time.Parse(globals.TimeLayout, nows)
	if err != nil {
		return
	} else {
		rt = inTimeSpan(start, end, ns)
	}
	return
}

type fn func()

func Cronos(f fn, hour int, interval int, rst chan interface{}) {
	defer func() {
		if r := recover(); r != nil {
			Cronos(f, hour, interval, rst)
		}
	}()
	waiting := true
	for {
		select {
		case <-rst:
			return
		case <-time.After(time.Duration(interval) * time.Minute):
			if waiting {
				if time.Now().Hour() == hour {
					f()
					waiting = false
				}
			} else {
				if time.Now().Hour() > hour {
					waiting = true
					println("waiting again")
				} else {
					println("not waiting")
				}
			}
		}
	}
}
