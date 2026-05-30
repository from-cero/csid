package redis

import "errors"

var (
	// ErrNoNodeAvailable is returned when all node ID slots (0...maxNodeID) are occupied in Redis.
	ErrNoNodeAvailable = errors.New("no node ID available")

	// ErrNotAcquired is returned when Release is called before a successful Acquire.
	ErrNotAcquired = errors.New("release before acquire")

	// ErrOwnershipLost is returned when the heartbeat detects the node key was deleted or taken by another instance.
	ErrOwnershipLost = errors.New("node ID ownership lost")

	// ErrNilClient is returned when NewRegistry is called with a nil Redis client.
	ErrNilClient = errors.New("redis client cannot be nil")

	// ErrInvalidTTLConfig is returned when ttl is not greater than 3x heartbeat interval.
	ErrInvalidTTLConfig = errors.New("ttl must be greater than 3x heartbeat interval")

	// ErrInvalidMaxNodeID is returned when maxNodeID is negative.
	ErrInvalidMaxNodeID = errors.New("maxNodeID must be non-negative")

	// ErrAcquireInProgress is returned when Acquire is called while a prior Acquire is still in flight.
	ErrAcquireInProgress = errors.New("acquire already in progress")
)
