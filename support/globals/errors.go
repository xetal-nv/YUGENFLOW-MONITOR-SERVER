package globals

import "errors"

var (
	KeyInvalid    = errors.New("invalid key")
	SensorDBError = errors.New("diskCache operation failed")
	Error         = errors.New("operation failed")
)
