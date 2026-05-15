package crid

import "errors"

var (
	ErrClockBackward     = errors.New("crid: clock moved backward beyond tolerance")
	ErrWorkerExhausted   = errors.New("crid: all worker IDs are taken")
	ErrInvalidDatacenter = errors.New("crid: datacenter ID exceeds maximum")
	ErrInvalidWorkerID   = errors.New("crid: worker ID exceeds maximum")
	ErrInvalidEntity     = errors.New("crid: entity type exceeds maximum for this format")
	ErrNoRegistry        = errors.New("crid: Registry must be set; use WithRegistry()")
)
