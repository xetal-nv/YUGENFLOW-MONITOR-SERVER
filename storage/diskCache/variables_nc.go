// +build newcache

package diskCache

import "github.com/fpessolano/jac"

var (
	definitions, // sensor definitions
	lookup, // id to mac table
	activeDevices, // active devices
	invalidDevices, // active devices
	maliciousMac, // mac of malicious devices
	maliciousIp, // ip of malicious devices
	recovery, // saved recovery data
	shadowRecovery jac.Bucket // saved shadow recovery data
)
