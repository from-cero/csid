package csid

import "errors"

var (
	ErrInvalidConfig = errors.New("invalid config")

	// ErrInvalidBitFormat is returned when
	// format.timestampBits + format.nodeBits + format.sequenceBits + ... != 63.
	ErrInvalidBitFormat = errors.New("bit format must sum to 63")

	// ErrEpochInFuture is returned when
	// config.epoch is the time after the current system time at the time of Node creation.
	ErrEpochInFuture = errors.New("epoch in future")

	// ErrMaxClockDrift is returned when
	// config.maxClockDrift is negative.
	ErrMaxClockDrift = errors.New("maxClockDrift is negative")

	// ErrNilRegistry is returned when
	// New is called with a nil Registry.
	ErrNilRegistry = errors.New("registry cannot be nil")

	// ErrInvalidNodeID is returned when
	// the node ID acquired is out of range for the given format.nodeBits.
	ErrInvalidNodeID = errors.New("node id is out of range for the given format")

	// ErrNodeClosed is returned when
	// Node.Generate is called on a closed Node.
	ErrNodeClosed = errors.New("node is closed")

	// ErrClockBeforeEpoch is returned when
	// the system clock is behind the configured epoch.
	ErrClockBeforeEpoch = errors.New("system clock is before the configured epoch")

	// ErrTimestampOverflow is returned when
	// the current time exceeds the maximum representable timestamp for the given format.
	ErrTimestampOverflow = errors.New("timestamp exceeds maximum for the given format")

	// ErrClockBackward is returned when
	// the system clock drifts backward beyond config.maxClockDrift.
	// This error is retriable: the caller may call Generate again after a short wait.
	ErrClockBackward = errors.New("clock moved backward beyond tolerance")

	// ErrClockSyncFailed is returned when
	// the generator slept waiting for a backward clock to catch up, but the clock
	// was still behind after waking. This is transient and retriable.
	ErrClockSyncFailed = errors.New("clock failed to sync after waiting for backward drift")
)
