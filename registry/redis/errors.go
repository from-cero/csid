package redis

import "errors"

var (

	// ErrNoNodeAvailable is returned when all node ID slots (0...maxNodeID) are
	// occupied in Redis and Acquire cannot claim one.
	ErrNoNodeAvailable = errors.New("no node ID available: all slots are occupied")

	// ErrNotAcquired is returned when Release is called before a successful Acquire.
	ErrNotAcquired = errors.New("release called without a prior successful acquire")

	// ErrOwnershipLost is returned by the onHeartbeatFailure callback when the
	// heartbeat Lua script detects the key was deleted or overwritten by another instance.
	ErrOwnershipLost = errors.New("node ID ownership lost: key expired or taken by another instance")

	// ErrNilClient is returned when NewRegistry is called with a nil Redis client.
	ErrNilClient = errors.New("redis client cannot be nil")

	// ErrInvalidTTLConfig is returned when TTL is not greater than 3x heartbeat interval.
	ErrInvalidTTLConfig = errors.New("invalid redis registry config: ttl must be greater than 3x heartbeat interval")

	// ErrInvalidMaxNodeID is returned when maxNodeID is negative.
	ErrInvalidMaxNodeID = errors.New("maxNodeID must be >= 0")

	// ErrAcquireInProgress is returned when Acquire is called concurrently on the
	// same registry instance while a prior Acquire call is still in flight.
	ErrAcquireInProgress = errors.New("acquire already in progress on this registry instance")
)
