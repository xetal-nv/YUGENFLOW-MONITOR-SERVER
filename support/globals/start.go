package globals

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"strings"
)

func Start() {

	internalConfig, err := ini.InsensitiveLoad("gateserver.ini")
	if err != nil {
		fmt.Printf("Fail to read gateserver.ini file: %v", err)
		os.Exit(1)
	}
	Config, err = ini.InsensitiveLoad("configuration.ini")
	if err != nil {
		fmt.Printf("Fail to read configuration.ini file: %v", err)
		os.Exit(1)
	}
	AccessData, err = ini.InsensitiveLoad("access.ini")
	if err != nil {
		fmt.Printf("Fail to read access.ini file: %v", err)
		os.Exit(1)
	}

	// System configuration

	ChannellingLength = internalConfig.Section("buffers").Key("channelling").MustInt(5)
	SecurityLength = internalConfig.Section("buffers").Key("security").MustInt(50)

	Shadowing = internalConfig.Section("corrections").Key("shadowing").MustBool(false)
	AcceptNegatives = internalConfig.Section("corrections").Key("accept_errors").MustBool(false)

	SensorTimeout = internalConfig.Section("timeouts").Key("device").MustInt(5)
	ServerTimeout = internalConfig.Section("timeouts").Key("server").MustInt(5)
	MaliciousTimeout = internalConfig.Section("timeouts").Key("malicious").MustInt(120)
	ZombieTimeout = internalConfig.Section("timeouts").Key("zombie").MustInt(24)
	RepetitiveTimeout = internalConfig.Section("timeouts").Key("repetitive").MustInt(20)
	SettleTime = internalConfig.Section("timeouts").Key("shutdown").MustInt(3)

	SensorEEPROMResetDelay = internalConfig.Section("eeprom").Key("eeprom_delay").MustInt(10)
	SensorEEPROMResetStep = internalConfig.Section("eeprom").Key("eeprom_step").MustInt(3)
	SensorSettingsFile = internalConfig.Section("eeprom").Key("sensorEEPROM").MustString("")

	CRCused = internalConfig.Section("sensors").Key("crc_enabled").MustBool(true)
	fmt.Printf("*** WARNING: CRC usage is set to %v ***\n", CRCused)
	MaximumInvalidIDInternal = internalConfig.Section("sensors").Key("maximum_undefined_time").MustInt(5)
	ResetCloseTCP = internalConfig.Section("sensors").Key("reset_closure").MustBool(true)
	AsymmetryMax = internalConfig.Section("sensors").Key("asymmetry_max").MustInt(3)
	AsymmetryIter = internalConfig.Section("sensors").Key("asymmetry_iter").MustInt(5)
	if AsymmetryIter == 0 {
		fmt.Printf("*** INFO: gate asymmetry is disabled ***\n")
	} else {
		fmt.Printf("*** INFO: gate asymmetry is enabled (Max:%v, Iter:%v) ***\n", AsymmetryMax, AsymmetryIter)
	}
	AsymmetricNull = internalConfig.Section("sensors").Key("asymmetric_null").MustBool(false)
	AsymmetryReset = internalConfig.Section("sensors").Key("asymmetry_reset").MustInt(50)
	if AsymmetryReset < 50 {
		fmt.Printf("*** INFO: gate asymmetry reset provided %v is illegal, minimum value enforced ***\n", AsymmetryReset)
		AsymmetryReset = 50
	}
	ResetPeriod = internalConfig.Section("sensors").Key("reset_period").MustInt(20)
	ResetSlot = internalConfig.Section("sensors").Key("reset_slot").MustString("")
	if ResetSlot != "" {
		ResetSlot = strings.Trim(strings.Replace(ResetSlot, ".", ":", -1), "")
	}

	MaliciousTriesIP = internalConfig.Section("security").Key("maliciousIP_threshold").MustInt(50)
	MaliciousTriesMac = internalConfig.Section("security").Key("maliciousMAC_threshold").MustInt(5)
	MalicioudMode = internalConfig.Section("security").Key("malicious_control").MustInt(0)
	FailureThreshold = internalConfig.Section("security").Key("failure_threshold").MustInt(5)
	CRCMaliciousCount = internalConfig.Section("security").Key("CRC_errors_included").MustBool(false)
	if MalicioudMode == 2 {
		FailureThreshold = SevereFailureThreshold
	}
	EnforceStrict = internalConfig.Section("security").Key("enforce_strict").MustBool(false)

	SaveState = internalConfig.Section("shutdown").Key("save_state").MustBool(false)
	if SaveState {
		fmt.Printf("*** INFO: Server state is being preserved ***\n")
	}
	MaxStateAge = internalConfig.Section("shutdown").Key("state_max_age").MustInt(30)

	ResetChannel = make(chan string, ChannellingLength)

	// Access configuration
	TCPport = AccessData.Section("tcpserver").Key("tcp_port").MustString("3333")

	APIport = AccessData.Section("apiserver").Key("api_port").MustString("")
	DisableCORS = AccessData.Section("apiserver").Key("disable_cors").MustBool(false)

	DisableDatabase = AccessData.Section("database").Key("disable").MustBool(false)
	if DisableDatabase {
		fmt.Printf("*** WARNING: Database is disabled ***\n")
		APIport = ""
	}
	if APIport == "" {
		fmt.Printf("*** WARNING: API server is disabled ***\n")
	}

	if ExportEnabled {
		ExportAsync = AccessData.Section("export").Key("async").MustBool(false)
		ExportActualCommand = AccessData.Section("export").Key("current_command").MustString("")
		ExportActualArgument = AccessData.Section("export").Key("current_argument").MustString("")
		if ExportActualCommand != "" {
			fmt.Printf("*** WARNING: Actual data export enabled with command %v %v ***\n", ExportActualCommand,
				ExportActualArgument)
		}
		ExportReferenceCommand = AccessData.Section("export").Key("reference_command").MustString("")
		ExportReferenceArgument = AccessData.Section("export").Key("reference_argument").MustString("")
		if ExportReferenceCommand != "" {
			fmt.Printf("*** WARNING: Actual data export enabled with command %v %v ***\n", ExportReferenceCommand,
				ExportReferenceArgument)
		}
		if ExportAsync && (ExportActualCommand != "" || ExportReferenceCommand != "") {
			fmt.Printf("*** INFO: Export is set to ASYNCHRONOUS ***\n")
		} else {
			fmt.Printf("*** INFO: Export is set to SYNCHRONOUS ***\n")
		}
	}
}
