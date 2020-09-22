package dataformats

// basic flow data model used for data from sensors and gates
type SensorDefinition struct {
	Id      int  `json:"id"`
	Bypass  bool `json:"bypass"`
	Report  bool `json:"report"`
	Enforce bool `json:"enforce"`
	Strict  bool `json:"strict"`
}
