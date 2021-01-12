// +build newcache

package diskCache

// fixme the malicious caches seems to behave differently marking everything a malicious !!!

import (
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/jac"
	"github.com/fpessolano/mlogger"
	"gopkg.in/ini.v1"
	"os"
)

// read cache settings from cache.ini
func loadSettings() (options *jac.Options) {
	if internalConfig, err := ini.InsensitiveLoad("cache.ini"); err != nil {
		fmt.Printf("Fail to Get cache.ini file: %v", err)
		os.Exit(1)
	} else {
		options = &jac.Options{
			ExpirationTime:        internalConfig.Section("data").Key("expirationTime").MustInt(10080) * 60,
			IntervalConsolidation: internalConfig.Section("data").Key("intervalConsolidation").MustInt(1440) * 60,
			InternalBuffering:     internalConfig.Section("system").Key("internalBuffering").MustInt(10),
			LoadDelayMs:           internalConfig.Section("system").Key("loadDelayMs").MustInt(10),
			MaximumAge:            int64(internalConfig.Section("data").Key("maximumAge").MustInt(5) * 60),
			WorkingFolder:         internalConfig.Section("paths").Key("workingFolder").MustString(""),
			RecoveryFolder:        internalConfig.Section("paths").Key("recoveryFolder").MustString(""),
		}
		if err = os.MkdirAll(options.WorkingFolder, os.ModePerm); err != nil {
			//if err = os.MkdirAll(globals.DiskCachePath, os.ModePerm); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
		if err = os.MkdirAll(options.RecoveryFolder, os.ModePerm); err != nil {
			//if err = os.MkdirAll(globals.DiskCachePath, os.ModePerm); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	}
	return
}

func Start() error {
	var err error
	if globals.SensorDBLog, err = mlogger.DeclareLog("yugenflow_sensorDB", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_gsensorDB logfile.")
		os.Exit(0)
	}
	if err = mlogger.SetTextLimit(globals.SensorDBLog, 80, 20, 10); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	if err = jac.Initialise(false, loadSettings()); err != jac.IllegalParameter && err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	definitions, err = jac.NewBucket("definitions", jac.NoExpiration)
	if err == nil {
		lookup, err = jac.NewBucket("lookup", jac.NoExpiration)
	}
	if err == nil {
		activeDevices, err = jac.NewBucket("activeDevices", jac.NoExpiration)
	}
	if err == nil {
		invalidDevices, err = jac.NewBucket("invalidDevices", jac.DefaultExpiration)
	}
	if err == nil {
		maliciousMac, err = jac.NewBucket("maliciousMac", jac.DefaultExpiration)
	}
	if err == nil {
		maliciousIp, err = jac.NewBucket("maliciousIp", jac.DefaultExpiration)
	}
	if err == nil {
		recovery, err = jac.NewBucket("recovery", jac.NoExpiration)
	}
	if err == nil {
		shadowRecovery, err = jac.NewBucket("shadowRecovery", jac.NoExpiration)
	}
	if err != nil {
		mlogger.Panic(globals.SensorDBLog,
			mlogger.LoggerData{"diskCache.Start", "Error in opening buckets: " + err.Error(),
				[]int{0}, false}, true)
	}
	mlogger.Info(globals.SensorDBLog,
		mlogger.LoggerData{"diskCache.Start", "service started",
			[]int{1}, true})

	fmt.Println("*** INFO: SensorDB initialised ***")
	return nil
}

func Close() {
	fmt.Println("Closing SensorDB")
	definitions.Close(false)
	lookup.Close(false)
	activeDevices.Close(false)
	invalidDevices.Close(false)
	maliciousMac.Close(false)
	maliciousIp.Close(false)
	recovery.Close(false)
	shadowRecovery.Close(false)
	jac.Terminate()
	mlogger.Info(globals.SensorDBLog,
		mlogger.LoggerData{"diskCache.Start", "service stopped",
			[]int{1}, true})
}
