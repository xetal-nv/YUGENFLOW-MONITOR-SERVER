package codings

import (
	"fmt"
	"testing"
)

func Test_crc8(t *testing.T) {
	a := []byte{'1', '2', 'a'}
	fmt.Printf("%X\n", Crc8(a))
}
