package registers

import "testing"

func Test_Setup(t *testing.T) {
	TimedIntDBSSetUp()
	TimedIntDBSClose()
}
