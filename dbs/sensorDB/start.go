package sensorDB

import (
	"errors"
	"fmt"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
)

var main *bolt.DB

const (
	definitions   = "definitions"   // sensor definitions
	lookup        = "lookup"        // id to mac table
	activeDevices = "activeDevices" // active devices
	maliciousMac  = "maliciousMac"  // mac of malicious devices
	maliciousIp   = "maliciousIp"   // ip of malicious devices
)

func Start() {
	var err error
	if globals.SensorDBLog, err = mlogger.DeclareLog("yugenflow_gsensorDB", false); err != nil {
		fmt.Println("Fatal Error: Unable to set yugenflow_gsensorDB logfile.")
		os.Exit(0)
	}
	if err = mlogger.SetTextLimit(globals.SensorDBLog, 80, 20, 10); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if err = os.MkdirAll(globals.DiskCachePath, os.ModePerm); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	if main, err = bolt.Open(filepath.Join(globals.DiskCachePath, "devicetable.db"), 0600, nil); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	//noinspection GoNilness
	if err := main.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(definitions))
		if err != nil {
			return errors.New("could not create " + definitions + " bucket: " + err.Error())
		}
		_, err = tx.CreateBucketIfNotExists([]byte(lookup))
		if err != nil {
			return errors.New("could not create " + lookup + " bucket: " + err.Error())
		}
		_, err = tx.CreateBucketIfNotExists([]byte(activeDevices))
		if err != nil {
			return errors.New("could not create " + activeDevices + " bucket: " + err.Error())
		}
		_, err = tx.CreateBucketIfNotExists([]byte(maliciousMac))
		if err != nil {
			return errors.New("could not create " + maliciousMac + " bucket: " + err.Error())
		}
		_, err = tx.CreateBucketIfNotExists([]byte(maliciousIp))
		if err != nil {
			return errors.New("could not create " + maliciousIp + " bucket: " + err.Error())
		}
		return nil
	}); err != nil {
		mlogger.Panic(globals.SensorDBLog,
			mlogger.LoggerData{"sensorDB.Start", "Error in opening buckets: " + err.Error(),
				[]int{}, false}, true)
	}
	mlogger.Info(globals.SensorDBLog,
		mlogger.LoggerData{"sensorDB.Start", "service started",
			[]int{1}, true})

	fmt.Println("SensorDB initialised")
}

func Close() {
	_ = main.Close()
	fmt.Println("SensorDB closed")
	mlogger.Info(globals.SensorDBLog,
		mlogger.LoggerData{"sensorDB.Start", "service stopped",
			[]int{1}, true})
}
