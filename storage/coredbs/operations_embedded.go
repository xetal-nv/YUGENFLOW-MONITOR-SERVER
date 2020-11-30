// +build embedded

package coredbs

import (
	"gateserver/dataformats"
)

func SaveSpaceData(nd dataformats.SpaceState) error {
	return nil
}

func SaveReferenceData(nd dataformats.MeasurementSample) error {
	return nil
}

func SaveShadowSpaceData(nd dataformats.SpaceState) error {
	return nil
}

func ReadSpaceData(spacename string, howMany int) (result []dataformats.MeasurementSample, err error) {
	return
}

func ReadReferenceData(spacename string, howMany int) (result []dataformats.MeasurementSample, err error) {
	return
}

func ReadReferenceDataSeries(spacename string, ts0, ts1 int) (result []dataformats.MeasurementSample, err error) {
	return
}

func ReadSpaceDataSeries(spacename string, ts0, ts1 int) (result []dataformats.MeasurementSample, err error) {
	return
}

func VerifyPresence(spacename string, ts0, ts1 int) (present bool, err error) {
	return
}
