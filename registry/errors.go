package registry

import (
	"errors"
)

var (
	// ErrEmptyEnvNodeID is returned when
	// the NODE_ID environment variable is not set.
	ErrEmptyEnvNodeID = errors.New("NODE_ID is not set in environment variables")

	// ErrInvalidEnvNodeID is returned when
	// the NODE_ID environment variable is not a valid integer.
	ErrInvalidEnvNodeID = errors.New("NODE_ID environment variable is not a valid integer")
)
