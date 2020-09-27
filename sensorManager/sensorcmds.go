package sensorManager

import (
	"encoding/binary"
	"fmt"
	"gateserver/support/globals"
	"net"
	"time"
)

func setID(chs SensorChannel, id int) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(id))

	cmd := []byte{cmdAPI["setid"].cmd, b[6], b[7]}
	if globals.DebugActive {
		fmt.Printf("sensor setID to be done %v:%x\n", id, cmd)

	}
	if chs.Tcp == nil || chs.CmdAnswer == nil || chs.Commands == nil || chs.Reset == nil {
		return globals.Error
	}
	select {
	case chs.Commands <- cmd:
		go func() {
			select {
			case res := <-chs.Commands:
				if globals.DebugActive {
					if res != nil {
						fmt.Printf("sensor setID done %v:%x\n", id, cmd)
					} else {
						fmt.Printf("sensor setID failed %v:%x\n", id, cmd)
					}
				}
			case <-time.After(time.Duration(globals.ZombieTimeout) * time.Hour):
			}
		}()
		return nil
	case <-time.After(time.Duration(globals.SensorTimeout) * time.Second):
	}
	return globals.Error
}

// TODO to be done
func refreshEEPROM(conn net.Conn, mach string) error {
	println("sensor EEPROM refresh to be done")
	return nil
}
