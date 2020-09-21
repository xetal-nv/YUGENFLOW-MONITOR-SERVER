package globals

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
)

func Start() {

	_, err1 := ini.InsensitiveLoad("gateserver.ini")
	//cfg, err := ini.InsensitiveLoad("yfserver.ini")
	if err1 != nil {
		fmt.Printf("Fail to read gateserver.ini file: %v", err1)
		os.Exit(1)
	}
	cfg, err2 := ini.InsensitiveLoad("configuration.ini")
	//cfg, err := ini.InsensitiveLoad("yfserver.ini")
	if err2 != nil {
		fmt.Printf("Fail to read configuration.ini file: %v", err2)
		os.Exit(1)
	}

	//Timeout = cfg.Section("server").Key("timeout").MustInt(20)
	//ServerPort = cfg.Section("server").Key("port").MustInt(8891)
	//BufferLength = cfg.Section("server").Key("buffer_length").MustInt(100)
	//DisableCORS = cfg.Section("server").Key("disable_cors").MustBool(false)
	//APIServer = cfg.Section("server").Key("api_server").MustString("")
	//if APIServer == "" {
	//	fmt.Println("Error in ini file, API server address is empty")
	//	os.Exit(0)
	//}
	//
	//IdentityLife = cfg.Section("cache").Key("identifier_life").MustInt(7)
	//LinkLife = cfg.Section("cache").Key("link_life").MustInt(5)
	//
	//TemporaryLinkLength = cfg.Section("definitions").Key("link_length").MustInt(16)
	//CommandServer = cfg.Section("definitions").Key("command_server").MustString("")
	//if CommandServer == "" {
	//	fmt.Println("Error in ini file, command server address is empty")
	//	os.Exit(0)
	//}
	//SupportEmail = cfg.Section("notifications").Key("support_email").MustString("")
	//TemplateLink = cfg.Section("notifications").Key("link_template").MustInt(1673099)

	for _, b := range cfg.Section("gates").KeyStrings() {
		fmt.Println(b, cfg.Section("gates").Key(b))
	}

}
