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
	Qualifier string                `json:"qualifier"`
	Space     string                `json:"space"`
	Ts        int64                 `json:"timestamp"`
	Val       float64               `json:"value"`
	FlowIn    int                   `json:"in,omitempty"`
	FlowOut   int                   `json:"out,omitempty"`
	Flows     map[string]EntryState `json:"flows"`
}
