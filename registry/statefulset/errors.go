package statefulset

import "errors"

var (
	// ErrInvalidMaxNodeID is returned when maxNodeID is negative.
	ErrInvalidMaxNodeID = errors.New("maxNodeID must be >= 0")

	// ErrInvalidHostname is returned when the pod name does not contain a parseable
	// non-negative integer ordinal as its last dash-separated segment.
	ErrInvalidHostname = errors.New("cannot parse StatefulSet ordinal from pod name")

	// ErrOrdinalOutOfRange is returned when the parsed ordinal exceeds maxNodeID.
	ErrOrdinalOutOfRange = errors.New("StatefulSet ordinal exceeds maxNodeID")

	// ErrNotAcquired is returned when Release is called before a successful Acquire.
	ErrNotAcquired = errors.New("release called without a prior successful acquire")
)
