package ceroid

import "errors"

var (
	// ErrFormatTooManyBits is returned when TimestampBits + NodeBits + SequenceBits != 63.
	ErrFormatTooManyBits = errors.New("format bits must sum to 63")
	// ErrInvalidNodeID is returned when the acquired node ID exceeds the maximum allowed by the format.
	ErrInvalidNodeID = errors.New("node ID is out of range for the given format")
	// ErrClockBackward is returned when the system clock drifts backward beyond MaxClockDrift.
	ErrClockBackward = errors.New("clock moved backward beyond tolerance")
	// ErrNilRegistry is returned when NewNode is called with a nil Registry.
	ErrNilRegistry = errors.New("registry cannot be nil")
	// ErrClosed is returned when Generate is called on a closed Node.
	ErrClosed = errors.New("node is closed")
)
