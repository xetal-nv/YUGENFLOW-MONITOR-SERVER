package support

import (
	"testing"
)

func Test_DLogger(t *testing.T) {
	setUpDevLogger()
	DLog <- DevData{"pippo", Timestamp(), "this is just a test", []int{1, 0, 1}, false}
	// remember to set chan buffer to 1
	DLog <- DevData{Tag: "skip"}
}
