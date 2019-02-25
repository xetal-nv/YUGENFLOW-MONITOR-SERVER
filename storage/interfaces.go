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

// All types beinf transmitted via registers must implement this interface
type GenericData interface {
	Extract(interface{}) error
}
