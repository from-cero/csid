package crid

import "fmt"

// Format defines the bit layout of a 63-bit Snowflake ID.
// All field widths must sum to exactly 63. EnvBits must be 0 or 1.
type Format struct {
	TimestampBits  uint8
	EntityBits     uint8
	DatacenterBits uint8
	EnvBits        uint8
	WorkerBits     uint8
	SequenceBits   uint8
}

// DefaultFormat is the standard Cero ID bit layout.
var DefaultFormat = Format{
	TimestampBits:  39,
	EntityBits:     4,
	DatacenterBits: 4,
	EnvBits:        1,
	WorkerBits:     7,
	SequenceBits:   8,
}

// Validate reports whether f is a legal format.
func (f Format) Validate() error {
	if f.EnvBits > 1 {
		return fmt.Errorf("ceroid: EnvBits must be 0 or 1, got %d", f.EnvBits)
	}
	sum := int(f.TimestampBits) + int(f.EntityBits) + int(f.DatacenterBits) +
		int(f.EnvBits) + int(f.WorkerBits) + int(f.SequenceBits)
	if sum != 63 {
		return fmt.Errorf("ceroid: format bits must sum to 63, got %d", sum)
	}
	return nil
}

// compiled holds pre-computed shifts and masks derived from a Format.
type compiled struct {
	shiftTimestamp  uint8
	shiftEntity     uint8
	shiftDatacenter uint8
	shiftEnv        uint8
	shiftWorker     uint8

	maskTimestamp  int64
	maskEntity     int64
	maskDatacenter int64
	maskEnv        int64
	maskWorker     int64
	maskSequence   int64

	maxSeq    int64
	maxWorker int64
	maxDC     int64
	maxEntity int64
}

func compileFormat(f Format) compiled {
	sw := f.SequenceBits
	se := sw + f.WorkerBits
	sd := se + f.EnvBits
	sn := sd + f.DatacenterBits
	st := sn + f.EntityBits

	mask := func(bits uint8) int64 {
		if bits == 0 {
			return 0
		}
		return (int64(1) << bits) - 1
	}

	return compiled{
		shiftTimestamp:  st,
		shiftEntity:     sn,
		shiftDatacenter: sd,
		shiftEnv:        se,
		shiftWorker:     sw,

		maskTimestamp:  mask(f.TimestampBits),
		maskEntity:     mask(f.EntityBits),
		maskDatacenter: mask(f.DatacenterBits),
		maskEnv:        mask(f.EnvBits),
		maskWorker:     mask(f.WorkerBits),
		maskSequence:   mask(f.SequenceBits),

		maxSeq:    mask(f.SequenceBits),
		maxWorker: mask(f.WorkerBits),
		maxDC:     mask(f.DatacenterBits),
		maxEntity: mask(f.EntityBits),
	}
}

var defaultCompiled = compileFormat(DefaultFormat)
