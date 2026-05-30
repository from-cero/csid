package statefulset

import "errors"

var (
	// ErrInvalidPodName is returned when the pod name has no parseable non-negative ordinal after the last dash.
	ErrInvalidPodName = errors.New("cannot parse StatefulSet ordinal from pod name")

	// ErrNotAcquired is returned when Release is called before a successful Acquire.
	ErrNotAcquired = errors.New("release before acquire")
)
