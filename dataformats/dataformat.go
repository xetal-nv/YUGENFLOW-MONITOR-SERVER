package dataformats

type Commanding []byte

// sensor data model
type SensorDefinition struct {
	Id        int  `json:"id"`
	Bypass    bool `json:"bypass"`
	Report    bool `json:"report"`
	Enforce   bool `json:"enforce"`
	Strict    bool `json:"strict"`
	Reversed  bool `json:"reversed"`
	Suspected int  `json:"numberMarkings"`
	Disabled  bool `json:"disabled"`
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
	Id  string `json:"id"`
	In  int    `json:"in"`
	Out int    `json:"out"`
}

// entry flow data model used for database storage
type EntryState struct {
	Id       string          `json:"id"`
	Ts       int64           `json:"ts"`
	Count    int             `json:"netflow"`
	State    bool            `json:"-"`
	Reversed bool            `json:"reversed"`
	Flows    map[string]Flow `json:"flows"`
}

// space flow data model used for database storage
type SpaceState struct {
	Id    string                `json:"id"`
	Ts    int64                 `json:"ts"`
	Count int                   `json:"netflow"`
	State bool                  `json:"-"`
	Flows map[string]EntryState `json:"flows"`
}
