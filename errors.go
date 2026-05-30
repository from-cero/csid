package csid

import "errors"

var (
	// ErrInvalidConfig is returned when config validation fails; it wraps the specific validation errors.
	ErrInvalidConfig = errors.New("invalid config")

	// ErrInvalidBitFormat is returned when the bit layout does not sum to 63.
	ErrInvalidBitFormat = errors.New("bit format must sum to 63")

	// ErrEpochInFuture is returned when cfg.epoch is the time after the current system time at the time of Node creation.
	ErrEpochInFuture = errors.New("epoch in future")

	// ErrMaxClockDrift is returned when config.maxClockDrift is negative.
	ErrMaxClockDrift = errors.New("maxClockDrift is negative")

	// ErrNilRegistry is returned when New is called with a nil Registry.
	ErrNilRegistry = errors.New("registry cannot be nil")

	// ErrInvalidNodeID is returned when the acquired node ID exceeds the range allowed by format.nodeBits.
	ErrInvalidNodeID = errors.New("node ID out of range")

	// ErrNodeClosed is returned when Node.Generate is called on a closed Node.
	ErrNodeClosed = errors.New("node is closed")

	// ErrClockBeforeEpoch is returned when the system clock is behind the configured epoch.
	ErrClockBeforeEpoch = errors.New("system clock is before the configured epoch")

	// ErrTimestampOverflow is returned when the timestamp exceeds the maximum for the given format.
	ErrTimestampOverflow = errors.New("timestamp overflow")

	// ErrClockBackward is returned when the system clock drifts backward beyond config.maxClockDrift.
	ErrClockBackward = errors.New("clock moved backward beyond tolerance")

	// ErrClockSyncFailed is returned when the clock was still behind after sleeping for backward drift recovery.
	ErrClockSyncFailed = errors.New("clock sync failed after backward drift")
)
