package support

import "os"

type fn func()

// RunWithRecovery runs function f() and in case of panic
// it executed first function r() and restart itself
func RunWithRecovery(f, r fn) {
	defer func() {
		if e := recover(); e != nil {
			if r != nil {
				r()
			}
			go RunWithRecovery(f, r)
		}
	}()
	f()
}

// fileExists checks if a file exists and is not a directory before we

// try using it to prevent further errors.

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// RunWithPanicCheck runs function f() and in case of panic
// it executed first function r() and returns false
// otherwise it returns true
func RunWithPanicCheck(f, r fn) (ok bool) {
	ok = true
	defer func() {
		if e := recover(); e != nil {
			if r != nil {
				r()
				ok = false
			}
		}
	}()
	f()
	return
}
