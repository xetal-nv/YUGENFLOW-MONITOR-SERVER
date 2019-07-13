package support

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// provide the time now as timestamp in ms
func Timestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// true if s includes e
func Contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// true if string a ends with string b
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

//  integer absolute
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// integer min, x<y
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// reduce string a to n characters
// or extend string a to n characters adding "_" at the end
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
	ns, err = time.Parse(TimeLayout, nows)
	if err != nil {
		return
	} else {
		rt = inTimeSpan(start, end, ns)
	}
	return
}

// used by InClosureTime
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

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	//noinspection GoUnhandledErrorResult
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

// Get the root folder of the executable
func GetCurrentExecDir() (dir string, err error) {
	path, err := exec.LookPath(os.Args[0])
	if err != nil {
		fmt.Printf("exec.LookPath(%s), err: %s\n", os.Args[0], err)
		return "", err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("filepath.Abs(%s), err: %s\n", path, err)
		return "", err
	}

	dir = filepath.Dir(absPath)
	return dir, nil

}

// provides the difference in ms between to hours provided as strings in the format hh:mm
// when start is later than end, it returns 0
func TimeDifferenceInSecs(start, end string) (int64, error) {
	if _, e := time.Parse(TimeLayout, start); e != nil {
		return 0, e
	}
	if _, e := time.Parse(TimeLayout, end); e != nil {
		return 0, e
	}
	t0 := strings.Split(strings.Trim(start, " "), ":")
	t1 := strings.Split(strings.Trim(end, " "), ":")
	if h0, e := strconv.Atoi(t0[0]); e == nil {
		if m0, e := strconv.Atoi(t0[1]); e == nil {
			if h1, e := strconv.Atoi(t1[0]); e == nil {
				if m1, e := strconv.Atoi(t1[1]); e == nil {
					t0s := (h0*3600 + m0*60) * 1000
					t1s := (h1*3600 + m1*60) * 1000
					if t0s > t1s {
						return 0, errors.New("End later than start")
					}
					return int64(t1s - t0s), nil
				}
			}
		}
	}
	return 0, errors.New("Invalid data")
}
