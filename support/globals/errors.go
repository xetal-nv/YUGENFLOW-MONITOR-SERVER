package globals

import "errors"

var (
	KeyInvalid       = errors.New("invalid key")
	SensorDBError    = errors.New("diskCache operation failed")
	Error            = errors.New("operation failed")
	InvalidOperation = errors.New("invalid operation")
	PartialError     = errors.New("errors have occurred")
	SyntaxError      = errors.New("syntaxt error")
)
