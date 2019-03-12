package support

import (
	"fmt"
	"testing"
)

func Test_DLogger(t *testing.T) {
	setUpDevLogger()
	DLog <- DevData{"check", Timestamp(), "this is just a test", []int{1, 2, 1}, false}
	// remember to set chan buffer to 1
	DLog <- DevData{Tag: "read"}
	fmt.Println(<-ODLog)
	//time.Sleep(5 * time.Second)
}
