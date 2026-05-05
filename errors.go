package ceroflake

import "errors"

var (
	// ErrClockBackward is returned when the wall clock drifts backward
	// beyond the configured tolerance and the generator refuses to block.
	ErrClockBackward = errors.New("ceroflake: clock moved backward beyond tolerance")

	// ErrWorkerIDExhausted is returned by the registry when all 128 worker
	// slots for the given datacenter/environment are already claimed.
	ErrWorkerIDExhausted = errors.New("ceroflake: all worker IDs are taken")

	// ErrInvalidDatacenter is returned when the datacenter ID exceeds 15.
	ErrInvalidDatacenter = errors.New("ceroflake: datacenter ID exceeds max value 15")

	// ErrInvalidWorkerID is returned when the worker ID exceeds 127.
	ErrInvalidWorkerID = errors.New("ceroflake: worker ID exceeds max value 127")
)
