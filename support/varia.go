package support

import (
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
