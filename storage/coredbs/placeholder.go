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
func SaveEntryData(nd dataformats.EntryState) {
	fmt.Printf("TBD: Store entry data %+v\n", nd)
}

// TODO this is just a placeholder
func SaveSpaceData(nd dataformats.SpaceState) {
	fmt.Printf("TBD: Store space data %+v\n", nd)
}

// TODO this is just a placeholder
func SaveEntryState(entryName string, nd dataformats.EntryState) error {
	fmt.Printf("TBD: Store entry %v state %+v\n", entryName, nd)
	return nil
}

// TODO this is just a placeholder
func LoadEntryState(entryName string) (dataformats.EntryState, error) {
	fmt.Printf("TBD: Load entry %v state\n", entryName)
	return dataformats.EntryState{}, nil
}

// TODO this is just a placeholder
func SaveSpaceState(entryName string, nd dataformats.SpaceState) error {
	fmt.Printf("TBD: Store space %v state %+v\n", entryName, nd)
	return nil
}

// TODO this is just a placeholder
func LoadSpaceState(entryName string) (dataformats.SpaceState, error) {
	fmt.Printf("TBD: Load space %v state\n", entryName)
	return dataformats.SpaceState{}, nil
}
