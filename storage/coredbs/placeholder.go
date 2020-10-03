package coredbs

import (
	"fmt"
	"gateserver/dataformats"
)

// TODO this is just a placeholder
func SaveGateData(nd dataformats.FlowData) {
	fmt.Printf("TBD: Store gate data %+v\n", nd)
}

// TODO this is just a placeholder
func SaveEntryData(nd dataformats.Entrydata) {
	fmt.Printf("TBD: Store entry data %+v\n", nd)
}

// TODO this is just a placeholder
func SaveEntryState(entryName string, nd dataformats.Entrydata) error {
	fmt.Printf("TBD: Store entry %v state %+v\n", entryName, nd)
	return nil
}

// TODO this is just a placeholder
func LoadEntryState(entryName string) (dataformats.Entrydata, error) {
	fmt.Printf("TBD: Load entry %v state\n", entryName)
	return dataformats.Entrydata{}, nil
}
