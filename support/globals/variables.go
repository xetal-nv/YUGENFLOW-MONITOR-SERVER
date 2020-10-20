package globals

import "gopkg.in/ini.v1"

// hardcoded parameters

const (
	VERSION       = "2.0.0"
	SEVERE        = 2
	NORMAL        = 1
	OFF           = 0
	TimeLayout    = "15:04" // time layout used to read the configuration file
	TimeLayoutDot = "15.04" // time layout used to read the configuration file
)

// this channel is used to reset a sensor (sends the MAC)
var ResetChannel chan string

// ini files
var Config, AccessData *ini.File

// logFiles
var SevereFailureThreshold, SensorManagerLog, GateManagerLog, SensorDBLog, EntryManagerLog,
	SpaceManagerLog, DBSLog, AvgsManagerLog, ApiManager, ExportManager int

// Parameters configurable via ini files
//noinspection GoExportedOwnDeclaration
var DebugActive, CRCused, SensorEEPROMResetEnabled, CRCMaliciousCount, EnforceStrict, AsymmetricNull,
	SaveState, Shadowing, AcceptNegatives, ResetCloseTCP, DisableDatabase, DisableCORS, EchoMode bool
var ChannellingLength, SettleTime, SensorTimeout, TCPdeadline, MaliciousTimeout, MaliciousTriesIP, ServerTimeout,
	MaliciousTriesMac, MalicioudMode, FailureThreshold, MaximumInvalidIDInternal, ZombieTimeout,
	RepetitiveTimeout, SecurityLength, SensorEEPROMResetDelay, SensorEEPROMResetStep,
	AsymmetryMax, AsymmetryIter, ResetPeriod, AsymmetryReset, MaxStateAge int
var APIport, TCPport, DiskCachePath, SensorSettingsFile, ResetSlot, DBpath, DBUser, DBUserPassword,
	ExportActualCommand, ExportActualArgument, ExportReferenceCommand, ExportReferenceArgument string
