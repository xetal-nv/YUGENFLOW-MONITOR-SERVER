// +build embedded

package coredbs

import "gateserver/support/globals"

func Start() (err error) {
	globals.DisableDatabase = true
	return
}

func Disconnect() error {
	return nil
}
