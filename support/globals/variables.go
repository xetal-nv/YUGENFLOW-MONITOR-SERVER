package globals

import "gopkg.in/ini.v1"

// hardcoded parameters

const (
	VERSION = "2.0.0"
)

// ini files
var Config *ini.File

// logFiles
var SensorManagerLog, DeviceManagerLog int

// Parameters configurable via ini files
//noinspection GoExportedOwnDeclaration
var DebugActive, CRCused, SensorEEPROMResetEnabled bool
var ChannellingLength, ShutdownTime, SensorTimeout, TCPdeadline int
var APIport, TCPport string
