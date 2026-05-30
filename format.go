package csid

// FormatOption configures the bit layout of a 63-bit Snowflake ID.
// The total number of bits allocated for timestamp, node, and sequence must equal 63.
type FormatOption func(*format)

// WithTimestampBits configures the number of bits allocated for the timestamp in the ID.
func WithTimestampBits(bits uint8) FormatOption {
	return func(f *format) {
		f.timestampBits = bits
	}
}

// WithNodeBits configures the number of bits allocated for the node ID in the ID.
func WithNodeBits(bits uint8) FormatOption {
	return func(f *format) {
		f.nodeBits = bits
	}
}

// WithSequenceBits configures the number of bits allocated for the sequence number in the ID.
func WithSequenceBits(bits uint8) FormatOption {
	return func(f *format) {
		f.sequenceBits = bits
	}
}

type format struct {
	timestampBits uint8 // number of bits allocated for the timestamp (in ms since epoch)
	nodeBits      uint8 // number of bits allocated for the node id
	sequenceBits  uint8 // number of bits allocated for the sequence number
}

var defaultFormat = format{
	timestampBits: 41, // 69 years of timestamps in ms
	nodeBits:      12, // 4096 nodes
	sequenceBits:  10, // 1024 IDs/ms/node
}

func applyFormatOptions(opts []FormatOption) format {
	res := defaultFormat
	for _, opt := range opts {
		opt(&res)
	}
	return res
}

func (f *format) validate() error {
	sum := int(f.timestampBits) + int(f.nodeBits) + int(f.sequenceBits)
	if sum != 63 {
		return ErrInvalidBitFormat
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

func (f *format) compileFormat() compiledFormat {
	sn := f.sequenceBits
	st := sn + f.nodeBits
	mask := func(bits uint8) int64 {
		return (int64(1) << bits) - 1
	}
	return compiledFormat{
		shiftTimestamp: st,
		shiftNode:      sn,
		maxTimestamp:   mask(f.timestampBits),
		maxNode:        mask(f.nodeBits),
		maxSeq:         mask(f.sequenceBits),
	}
}
