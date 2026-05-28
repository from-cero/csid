package statefulset

import "errors"

var (
	// ErrInvalidPodName is returned when the pod name does not contain a parseable
	// non-negative integer ordinal as its last dash-separated segment.
	ErrInvalidPodName = errors.New("cannot parse StatefulSet ordinal from pod name")

	// ErrNotAcquired is returned when Release is called before a successful Acquire.
	ErrNotAcquired = errors.New("release called without a prior successful acquire")
)
