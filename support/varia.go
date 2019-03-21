package support

import (
	"strconv"
	"strings"
	"time"
)

func Timestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func Contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func Stringending(a, b string, trim string) bool {
	a = strings.Trim(a, trim)
	b = strings.Trim(b, trim)
	for i := 0; i < len(b); i++ {
		if a[len(a)-len(b)+i] != b[i] {
			return false
		}
	}
	return true
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func StringLimit(a string, n int) string {
	if len(a) < n {
		for i := len(a); i < n; i++ {
			a += "_"
		}
	} else {
		a = a[:n]
	}
	return a
}

func InClosureTime(start, end time.Time) (rt bool, err error) {
	if start == end {
		return false, nil
	}
	now := time.Now()
	//TimeLayout := "15:04"
	var ns time.Time
	ns, err = time.Parse(TimeLayout, strconv.Itoa(now.Hour())+":"+strconv.Itoa(now.Minute()))
	//srt, e2 := time.Parse(TimeLayout, start)
	//ed, e3 := time.Parse(TimeLayout, end)
	if err != nil {
		return
	} else {
		rt = inTimeSpan(start, end, ns)
	}
	return
}

func inTimeSpan(start, end, check time.Time) bool {
	//fmt.Println(start, end, check )
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
