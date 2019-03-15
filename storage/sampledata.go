package storage

// Interface for all type of data manageable as sample
type SampleData interface {
	Marshal() []byte
	Unmarshal([]byte) error
	Ts() int64
	Tag() string
	MarshalSize() int
	MarshalSizeModifiers() []int
	Extract(interface{}) error
	Valid() bool
	UnmarshalSliceSS(string, []int64, [][]byte) []SampleData
}
