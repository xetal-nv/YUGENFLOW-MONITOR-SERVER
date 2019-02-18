package support

import (
	"testing"
)

func Test_DLogger(t *testing.T) {
	SetUpDevLogger()
	DLog <- DevData{"pippo", Timestamp(), "this is just a test", []int{1, 0, 1}}
	DLog <- DevData{Tag: "skip"}
}
