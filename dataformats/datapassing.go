package dataformats

// basic flow data model used for data from sensors and gates
type FlowData struct {
	Type    string `json:"elementType"`
	Name    string `json:"name"`
	Id      int    `json:"id"`
	Ts      int64  `json:"timestamp"`
	Netflow int    `json:"netflow"`
}

type MeasurementSample struct {
	Qualifier string  `json:"qualifier"`
	Space     string  `json:"space"`
	Ts        int64   `json:"timestamp"`
	Val       float64 `json:"value"`
	//FlowIn         int                   `json:"in,omitempty"`
	//FlowOut        int                   `json:"out,omitempty"`
	//StartTimeFlows int64                 `json:"startTimeFlows,omitempty"`
	//TsOverflow     int64                 `json:"flowOverlowTs,omitempty"`
	Flows map[string]EntryState `json:"flows"`
}

type MeasurementSampleWithFlows struct {
	Qualifier      string                         `json:"qualifier"`
	Space          string                         `json:"space"`
	Ts             int64                          `json:"timestamp"`
	Count          float64                        `json:"count"`
	FlowIn         int                            `json:"in"`
	FlowOut        int                            `json:"out"`
	StartTimeFlows int64                          `json:"startTimeFlows"`
	TsOverflow     int64                          `json:"overflowTs"`
	Flows          map[string]EntryStateWithFlows `json:"flows"`
}
