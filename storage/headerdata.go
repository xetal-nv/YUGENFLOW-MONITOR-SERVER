package storage

import (
	"encoding/binary"
	"errors"
)

// Data is the data format for the header
type headerdata struct {
	id       string
	fromRst  uint64
	step     uint32
	lastUpdt uint64
	created  uint64
}

// Marshal encodes a Data values into codeddata
func (hd *headerdata) Marshal() []byte {
	r := make([]byte, 8)
	binary.LittleEndian.PutUint64(r, hd.fromRst)
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, hd.step)
	r = append(r, b...)
	c := make([]byte, 8)
	binary.LittleEndian.PutUint64(c, hd.lastUpdt)
	r = append(r, c...)
	binary.LittleEndian.PutUint64(c, hd.created)
	r = append(r, c...)
	return r
}

// Unmarshal decodes a codeddata into headerdata
func (hd *headerdata) Unmarshal(c []byte) error {

	if len(c) != 28 {
		return errors.New("Invalid raw data provided")
	}
	hd.fromRst = binary.LittleEndian.Uint64(c[0:8])
	hd.step = binary.LittleEndian.Uint32(c[8:12])
	hd.lastUpdt = binary.LittleEndian.Uint64(c[12:20])
	hd.created = binary.LittleEndian.Uint64(c[20:28])
	return nil

}
