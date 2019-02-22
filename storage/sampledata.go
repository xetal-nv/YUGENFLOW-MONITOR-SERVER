package storage

import "fmt"

// Interface for all type of data manageable as sample
type SampleData interface {
	Marshal() []byte
	Unmarshal([]byte) error
	Ts() int64
	Tag() string
	MarshalSize() int
	Extract(interface{}) error
}

func TestSampleDataCompliance(_ SampleData) {
	fmt.Println("Compliant")
}
