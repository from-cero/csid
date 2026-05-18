package ceroid

import "errors"

var (
	ErrFormatTooManyBits = errors.New("format bits exceed 63")
	ErrInvalidNodeID     = errors.New("node ID is out of range for the given format")
	ErrClockBackward     = errors.New("clock moved backward beyond tolerance")
	ErrNilRegistry       = errors.New("registry must be set; use WithRegistry()")
	ErrEnvMissingNodeID  = errors.New("NODE_ID is not set in environment variables")
)
