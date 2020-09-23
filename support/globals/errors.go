package globals

import "errors"

var (
	KeyInvalid    = errors.New("invalid key")
	SensorDBError = errors.New("sensorDB operation failed")
	//EmailError           = errors.New("device not fully registered")
	//EmailFailed          = errors.New("failed to send command link")
)
