package support

type fn func()

// RunWithRecovery runs function f() and in case of panic
// it executed first function r(=) and restart itself
func RunWithRecovery(f, r fn) {
	defer func() {
		if e := recover(); e != nil {
			r()
			go RunWithRecovery(f, r)
		}
	}()
	f()
}
