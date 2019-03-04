package servers

// All types beinf transmitted via registers must implement this interface
type GenericData interface {
	Extract(interface{}) error
	SetTag(string)
	SetVal(...int)
	SetTs(int64)
	//NewEl() GenericData
}
