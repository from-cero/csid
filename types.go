package ceroid

import "time"

type ID int64

type Options struct {
	Epoch    time.Time
	NodeID   int64
	NodeBits uint8
	StepBits uint8
}
