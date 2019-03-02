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
}

// All types beinf transmitted via registers must implement this interface
type GenericData interface {
	Extract(interface{}) error
	SetTag(string)
	SetVal(...int)
	SetTs(int64)
	NewEl() GenericData
}
