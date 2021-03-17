package sensorManager

import (
	"encoding/binary"
	"errors"
	"fmt"
	"gateserver/codings"
	"gateserver/support/globals"
	"github.com/fpessolano/mlogger"
	"math"
	"net"
	"time"
)

func setID(chs SensorChannel, id int) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(id))

	cmd := []byte{CmdAPI["setid"].Cmd, b[6], b[7]}
	if globals.DebugActive {
		fmt.Printf("sensor setID to be done %v:%x\n", id, cmd)

	}
	if chs.Tcp == nil || chs.CmdAnswer == nil || chs.Commands == nil || chs.reset == nil {
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

func refreshEEPROM(conn net.Conn, mac string) (err error) {

	// sends the command for a maximum of eepromResetTries times before reporting error
	sendCommand := func(command string, value uint32) (e error) {
		e = errors.New("server.setSensorParameters: Failed to execute command " + command)
		for i := 0; i < eepromResetTries; i++ {
			time.Sleep(time.Duration(globals.SensorEEPROMResetStep) * time.Second)
			if v, ok := CmdAPI[command]; ok {
				cmd := []byte{v.Cmd}
				bs := make([]byte, 4)
				binary.BigEndian.PutUint32(bs, value)
				cmd = append(cmd, bs[4-v.Lgt:4]...)
				cmd = append(cmd, codings.Crc8(cmd))
				if e := conn.SetWriteDeadline(time.Now().Add(time.Duration(globals.SensorTimeout) * time.Second)); e == nil {
					if _, e := conn.Write(cmd); e == nil {
					readLoop:
						// we give it a maximum of max (4, eepromResetTries) for the sensor to answer to the command
						for j := 0; j < int(math.Max(float64(4), float64(eepromResetTries))); j++ {
							ans := make([]byte, 1)
							if e := conn.SetReadDeadline(time.Now().Add(time.Duration(10*globals.SensorTimeout) *
								time.Second)); e == nil {
								if _, e := conn.Read(ans); e == nil {
									switch ans[0] {
									case 1:
										// sensor data, discard the rest of the message
										var data []byte
										if globals.CRCused {
											data = make([]byte, 4)
										} else {
											data = make([]byte, 3)
										}
										_, _ = conn.Read(data)
									case cmd[0]:
										// answer to command, discard CRC if present
										if globals.CRCused {
											_, _ = conn.Read(make([]byte, 1))
										}
										return nil
									default:
										// illegal answer
										break readLoop
									}
								}
							}
						}
					}
				}
				// reset the all deadlines
				_ = conn.SetDeadline(time.Time{})
			}
		}
		return
	}

	time.Sleep(time.Duration(globals.SensorEEPROMResetDelay) * time.Second)
	if specs, ok := sensorData[mac]; ok || commonSensorSpecs.savg != 0 {
		if !ok {
			specs = commonSensorSpecs
		}
		eLab := "("
		if e := sendCommand("srate", uint32(specs.srate)); e != nil {
			eLab += "srate "
		}
		if e := sendCommand("savg", uint32(specs.savg)); e != nil {
			eLab += "savg "
		}
		if e := sendCommand("bgth", uint32(math.Round(specs.bgth*16))); e != nil {
			eLab += "bgth "
		}
		if e := sendCommand("occth", uint32(math.Round(specs.occth*16))); e != nil {
			eLab += "occth "
		}
		if eLab != "(" {
			err = errors.New("Failed to execute commands " + eLab + ") for device " + mac)
		}
	} else {
		mlogger.Warning(globals.SensorManagerLog,
			mlogger.LoggerData{"sensorManager.SensorEEPROMResetEnabled mac: " + mac,
				"sensorData cache is corrupted",
				[]int{1}, true})
		err = errors.New("servers.SensorEEPROMResetEnabled: sensorData cache is corrupted for device " + mac)
	}
	return
}
