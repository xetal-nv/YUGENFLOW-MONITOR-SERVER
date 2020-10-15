package dataformats

// basic flow data model used for data from sensors and gates
type FlowData struct {
	Type    string `json:"elementType"`
	Name    string `json:"name"`
	Id      int    `json:"id"`
	Ts      int64  `json:"timestamp"`
	Netflow int    `json:"netflow"`
}

type SimpleSample struct {
	Qualifier string                `json:"qualifier"`
	Ts        int64                 `json:"timestamp"`
	Val       float64               `json:"value"`
	Flows     map[string]EntryState `json:"flows"`
}
