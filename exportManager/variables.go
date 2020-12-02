package exportManager

import "gateserver/dataformats"

var ExportActuals chan dataformats.MeasurementSampleWithFlows
var ExportReference chan dataformats.MeasurementSample
