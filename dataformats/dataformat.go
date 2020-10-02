package dataformats

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
type GateDefinition struct {
	Id        string `json:"id"`
	Reversed  bool   `json:"reversed"`
	Suspected int    `json:"numberMarkings"`
	Disabled  bool   `json:"disabled"`
}
