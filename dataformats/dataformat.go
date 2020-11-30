package dataformats

type Commanding []byte

// sensor data model
type SensorDefinition struct {
	Mac     string `json:"mac,omitempty"`
	Id      int    `json:"id"`
	MaxRate int64  `json:"sampleMaximumRateNS"`
	Bypass  bool   `json:"bypass"`
	Report  bool   `json:"report"`
	Enforce bool   `json:"enforce"`
	Strict  bool   `json:"strict"`
	//Reversed  bool   `json:"reversed"`
	Reversed bool `json:"-"`
	//Suspected int    `json:"numberMarkings"`
	Suspected int `json:"-"`
	//Disabled  bool   `json:"disabled"`
	Disabled bool `json:"-"`
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
	Id         string `json:"id"`
	Netflow    int    `json:"netflow"`
	TsOverflow int64  `json:"flowOverlowTs,omitempty"`
	Reversed   bool   `json:"reversed,omitempty"`
	FlowIn     int    `json:"in,omitempty"`
	FlowOut    int    `json:"out,omitempty"`
}

// entry flow data model used for database storage
type EntryState struct {
	Id         string          `json:"id"`
	Ts         int64           `json:"Ts"`
	Count      int             `json:"netflow"`
	TsOverflow int64           `json:"flowOverlowTs,omitempty"`
	FlowIn     int             `json:"in,omitempty"`
	FlowOut    int             `json:"out,omitempty"`
	State      bool            `json:"-"`
	Reversed   bool            `json:"reversed"`
	Flows      map[string]Flow `json:"flows"`
}

// space flow data model used for database storage
type SpaceState struct {
	Id         string                `json:"id"`
	Ts         int64                 `json:"Ts"`
	Count      int                   `json:"netflow"`
	TsOverflow int64                 `json:"flowOverlowTs,omitempty"`
	FlowIn     int                   `json:"in,omitempty"`
	FlowOut    int                   `json:"out,omitempty"`
	State      bool                  `json:"-"`
	Flows      map[string]EntryState `json:"flows"`
}
