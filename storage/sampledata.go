package storage

// Interface for all type of data manageable as sample
type SampleData interface {
	Marshal() []byte                                         // encode data in a storable binary format
	Unmarshal([]byte) error                                  // decode binary data into usable data
	Ts() int64                                               // returns the timestamp
	Tag() string                                             // returns the series tag of the data
	MarshalSize() int                                        // returns the fixed data size once marshaled
	MarshalSizeModifiers() []int                             // returns the data size once marshaled as offset plus variable part size
	Extract(interface{}) error                               // extract the data from a generic interface{}
	Valid() bool                                             // true if the data is valid, false otherwise
	UnmarshalSliceSS(string, []int64, [][]byte) []SampleData // decode a slice of binary data into a slice of usable data
}
