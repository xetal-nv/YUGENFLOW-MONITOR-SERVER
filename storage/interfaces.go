package storage

// Interface for all type of data manageable as sample
type SampleData interface {
	Marshal() []byte
	Unmarshal([]byte) error
	Ts() int64
	Tag() string
	MarshalSize() int
	Extract(interface{}) error
}

type GenericData interface {
	Extract(interface{}) error
}
