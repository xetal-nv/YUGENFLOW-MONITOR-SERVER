package globals

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
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

	ChannellingLength = internalConfig.Section("buffers").Key("channelling").MustInt(5)
	ShutdownTime = internalConfig.Section("buffers").Key("shutdown").MustInt(3)

	//for _, b := range Config.Section("sensors").KeyStrings() {
	//	fmt.Println(b, Config.Section("sensors").Key(b))
	//}
	//
	//for _, b := range Config.Section("gates").KeyStrings() {
	//	fmt.Println(b, Config.Section("gates").Key(b))
	//}
	//
	//for _, b := range Config.Section("entries").KeyStrings() {
	//	fmt.Println(b, Config.Section("entries").Key(b))
	//}
	//
	//for _, b := range Config.Section("spaces").KeyStrings() {
	//	fmt.Println(b, Config.Section("spaces").Key(b))
	//}

}
