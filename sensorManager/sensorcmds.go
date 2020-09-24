package sensorManager

import "net"

// TODO to be done
func refreshEEPROM(conn net.Conn, mach string) error {
	println("sensor EEPROM refresh to be done")
	return nil
}

// TODO to be done
//  sensor ID will be reprogrammed and data will be discarded till this operation is done
//  in case of timeout the sensor will be disconnected and marked suspicious (if applicable)
//  we need to also update the lookup db
func setID(conn net.Conn, id int) error {
	return nil
}
