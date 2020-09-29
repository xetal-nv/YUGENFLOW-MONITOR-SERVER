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

	APIport = internalConfig.Section("server").Key("http_port").MustString("8080")
	TCPport = internalConfig.Section("server").Key("tcp_port").MustString("3333")
	//fmt.Printf("API and TCP ports set to %s and %s respectively\n", APIport, TCPport)

	ChannellingLength = internalConfig.Section("buffers").Key("channelling").MustInt(5)
	SecurityLength = internalConfig.Section("buffers").Key("security").MustInt(50)
	ShutdownTime = internalConfig.Section("buffers").Key("shutdown").MustInt(3)

	SensorTimeout = internalConfig.Section("timeouts").Key("device").MustInt(5)
	MaliciousTimeout = internalConfig.Section("timeouts").Key("malicious").MustInt(120)
	ZombieTimeout = internalConfig.Section("timeouts").Key("zombie").MustInt(24)
	RepetitiveTimeout = internalConfig.Section("timeouts").Key("repetitive").MustInt(20)
	SensorEEPROMResetDelay = internalConfig.Section("timeouts").Key("eeprom_delay").MustInt(10)
	SensorEEPROMResetStep = internalConfig.Section("timeouts").Key("eeprom_step").MustInt(3)

	CRCused = internalConfig.Section("sensors").Key("crc_enabled").MustBool(true)
	fmt.Printf("*** WARNING: CRC usage is set to %v ***\n", CRCused)
	MaliciousTriesIP = internalConfig.Section("sensors").Key("maliciousIP_threshold").MustInt(50)
	MaliciousTriesMac = internalConfig.Section("sensors").Key("maliciousMAC_threshold").MustInt(5)
	MalicioudMode = internalConfig.Section("sensors").Key("malicious_control").MustInt(0)
	FailureThreshold = internalConfig.Section("sensors").Key("failure_threshold").MustInt(5)
	CRCMaliciousCount = internalConfig.Section("sensors").Key("CRC_errors_included").MustBool(false)
	MaximumInvalidIDInternal = internalConfig.Section("sensors").Key("maximum_undefined_time").MustInt(5)
	if MalicioudMode == 2 {
		FailureThreshold = SevereFailureThreshold
	}
	AsymmetryValue = internalConfig.Section("sensors").Key("asymmetry_value").MustInt(3)
	AsymmetryMax = internalConfig.Section("sensors").Key("asymmetry_max").MustInt(5)
	ResetPeriod = internalConfig.Section("sensors").Key("reset_period").MustInt(20)
	ResetSlot = internalConfig.Section("sensors").Key("reset_slot").MustString("")
	if ResetSlot != "" {
		ResetSlot = strings.Trim(strings.Replace(ResetSlot, ".", ":", -1), "")
	}

	SensorSettingsFile = internalConfig.Section("options").Key("sensorEEPROM").MustString("")
	EnforceStrict = internalConfig.Section("options").Key("enforce_strict").MustBool(false)

}
