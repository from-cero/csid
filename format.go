package csid

// Format defines the bit layout of a 63-bit Snowflake ID. [timestamp | node | sequence]
type Format struct {
	TimestampBits uint8 // number of bits allocated for the timestamp (in ms since epoch)
	NodeBits      uint8 // number of bits allocated for the node id
	SequenceBits  uint8 // number of bits allocated for the sequence number
}

// DefaultFormat is the default 63-bit layout: 41-bit timestamp, 12-bit node, 10-bit sequence.
var DefaultFormat = Format{
	TimestampBits: 41, // 69 years of timestamps in ms
	NodeBits:      12, // 4096 nodes
	SequenceBits:  10, // 1024 IDs/ms/node
}

func (f Format) validate() error {
	sum := int(f.TimestampBits) + int(f.NodeBits) + int(f.SequenceBits)
	if sum != 63 {
		return ConfigErrInvalidBitFormat
	}
	return nil
}

type compiledFormat struct {
	shiftTimestamp uint8
	shiftNode      uint8
	maxTimestamp   int64
	maxNode        int64
	maxSeq         int64
}

func (f Format) compileFormat() compiledFormat {
	sn := f.SequenceBits
	st := sn + f.NodeBits
	mask := func(bits uint8) int64 {
		return (int64(1) << bits) - 1
	}
	return compiledFormat{
		shiftTimestamp: st,
		shiftNode:      sn,
		maxTimestamp:   mask(f.TimestampBits),
		maxNode:        mask(f.NodeBits),
		maxSeq:         mask(f.SequenceBits),
	}
}
