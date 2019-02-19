package support

import "time"

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

func Stringending(a, b string) bool {
	for i := 0; i < len(b); i++ {
		if a[len(a)-len(b)+i] != b[i] {
			return false
		}
	}
	return true
}
