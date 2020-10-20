package sensormodels

var command = map[byte][]byte{
	[]byte("\x07")[0]: []byte("\x07\x07"),
	[]byte("\x09")[0]: []byte("\x09\x09"),
	[]byte("\x0b")[0]: []byte("\x0b\x0b"),
	[]byte("\x0d")[0]: []byte("\x0d\x0d"),
}

var cmdArgs = map[byte]int{
	[]byte("\x02")[0]: 1,
	[]byte("\x03")[0]: 1,
	[]byte("\x04")[0]: 2,
	[]byte("\x05")[0]: 2,
	[]byte("\x0e")[0]: 2,
	[]byte("\x0b")[0]: 2,
}
