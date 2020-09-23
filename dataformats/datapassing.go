package dataformats

// basic flow data model used for data from sensors and gates
type FlowData struct {
	Type    string `json:"elementType"`
	Name    string `json:"name"`
	Id      int    `json:"id"`
	Ts      int64  `json:"timestamp"`
	Netflow int    `json:"netflow"`
}

type CommandAnswer interface{} // TODO define this channel
