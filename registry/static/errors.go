package static

import "errors"

var (
	// ErrEnvNodeIDNotSet is returned when the environment variable for the node ID is not set or empty.
	ErrEnvNodeIDNotSet = errors.New("node ID env var not set")

	// ErrInvalidEnvNodeID is returned when the environment variable is not a valid non-negative integer.
	ErrInvalidEnvNodeID = errors.New("node ID env var is not a valid non-negative integer")
)
