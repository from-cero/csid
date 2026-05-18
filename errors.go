package ceroid

import "errors"

var (
	ErrFormatTooManyBits = errors.New("format bits exceed 63")
	ErrInvalidNodeID     = errors.New("node ID exceeds maximum")
	ErrClockBackward     = errors.New("clock moved backward beyond tolerance")

	// ErrNoRegistry        = errors.New("Registry must be set; use WithRegistry()")
)
