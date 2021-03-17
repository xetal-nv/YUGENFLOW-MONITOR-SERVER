package dataformats

type Commanding []byte

// sensor data model
type SensorDefinition struct {
	Mac       string `json:"mac,omitempty"`
	Id        int    `json:"id"`
	MaxRate   int64  `json:"sampleMaximumRateNS"`
	Bypass    bool   `json:"bypass"`
	Report    bool   `json:"report"`
	Enforce   bool   `json:"enforce"`
	Strict    bool   `json:"strict"`
	Reversed  bool   `json:"-"`
	Suspected int    `json:"-"`
	Disabled  bool   `json:"-"`
}

// gate data model
type GateState struct {
	Id        string `json:"id"`
	Reversed  bool   `json:"reversed"`
	Suspected int    `json:"numberMarkings"`
	Disabled  bool   `json:"disabled"`
}

// generic flow
type Flow struct {
	Id        string `json:"id"`
	Variation int    `json:"variation"`
	Reversed  bool   `json:"-"`
}

type FlowWithFlows struct {
	Id         string `json:"id"`
	Variation  int    `json:"-"`
	Netflow    int    `json:"netflow"`
	TsOverflow int64  `json:"overflowTs"`
	FlowIn     int    `json:"in"`
	FlowOut    int    `json:"out"`
}

// entry flow data model used for database storage
type EntryState struct {
	Id        string          `json:"id"`
	Ts        int64           `json:"Ts"`
	Variation int             `json:"variation"`
	State     bool            `json:"-"`
	Reversed  bool            `json:"-"`
	Flows     map[string]Flow `json:"flows"`
}

type EntryStateWithFlows struct {
	Id         string                   `json:"id"`
	Ts         int64                    `json:"Ts"`
	Variation  int                      `json:"-"`
	Netflow    int                      `json:"netflow"`
	TsOverflow int64                    `json:"overflowTs"`
	FlowIn     int                      `json:"in"`
	FlowOut    int                      `json:"out"`
	State      bool                     `json:"-"`
	Flows      map[string]FlowWithFlows `json:"flows"`
}

// space flow data model used for database storage
type SpaceState struct {
	Id    string                `json:"id"`
	Ts    int64                 `json:"Ts"`
	Count int                   `json:"netflow"`
	State bool                  `json:"-"`
	Reset bool                  `json:"-"`
	Flows map[string]EntryState `json:"flows"`
}
