package globals

import "gopkg.in/ini.v1"

// hardcoded parameters

const (
	VERSION = "2.0.0"
	SEVERE  = 2
	NORMAL  = 1
	OFF     = 0
)

// ini files
var Config *ini.File

// logFiles
var SevereFailureThreshold, SensorManagerLog, GateManagerLog, SensorDBLog int

// Parameters configurable via ini files
//noinspection GoExportedOwnDeclaration
var DebugActive, CRCused, SensorEEPROMResetEnabled, CRCMaliciousCount, EnforceStrict, AsymmetricNull bool
var ChannellingLength, ShutdownTime, SensorTimeout, TCPdeadline, MaliciousTimeout, MaliciousTriesIP,
	MaliciousTriesMac, MalicioudMode, FailureThreshold, MaximumInvalidIDInternal, ZombieTimeout,
	RepetitiveTimeout, SecurityLength, SensorEEPROMResetDelay, SensorEEPROMResetStep,
	AsymmetryMax, AsymmetryIter, ResetPeriod int
var APIport, TCPport, DiskCachePath, SensorSettingsFile, ResetSlot string
