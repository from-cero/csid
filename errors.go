package ceroid

import "errors"

var (
	ErrFormatTooManyBits = errors.New("format bits must sum to 63")
	ErrInvalidNodeID     = errors.New("node ID is out of range for the given format")
	ErrClockBackward     = errors.New("clock moved backward beyond tolerance")
	ErrNilRegistry       = errors.New("registry must be set; use WithRegistry()")
	ErrClosed            = errors.New("node is closed")
)
