package ceroid

import (
	"strconv"
	"time"
)

// Config holds all configuration for a Node or Parser.
type Config struct {
	Format        Format
	Epoch         time.Time
	MaxClockDrift time.Duration
}

// Option is a functional option for configuring a Node or Parser.
type Option func(*Config)

// WithFormat sets the bit layout for IDs. The default is DefaultFormat.
func WithFormat(f Format) Option { return func(c *Config) { c.Format = f } }

// WithEpoch sets the custom epoch used as the zero time for timestamps.
// The default epoch is 2026-01-01 00:00:00 UTC.
func WithEpoch(e time.Time) Option { return func(c *Config) { c.Epoch = e } }

// WithMaxClockDrift sets the maximum tolerated backward clock drift before
// Generate returns ErrClockBackward. The default is 10ms.
func WithMaxClockDrift(d time.Duration) Option { return func(c *Config) { c.MaxClockDrift = d } }

func applyOptions(opts []Option) Config {
	cfg := Config{
		Format:        DefaultFormat,
		Epoch:         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		MaxClockDrift: 10 * time.Millisecond,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// ID is a 63-bit Snowflake-style distributed identifier.
type ID int64

// Int64 returns the ID as a plain int64.
func (id ID) Int64() int64 {
	return int64(id)
}

// String returns the ID as a decimal string.
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// MarshalJSON encodes the ID as a quoted decimal string to avoid precision
// loss in JavaScript, which cannot represent 63-bit integers exactly.
func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.String() + `"`), nil
}

// ParsedID holds the decoded components of an ID.
type ParsedID struct {
	Timestamp time.Time
	Node      int64
	Sequence  int64
}

// String returns a human-readable representation of the parsed ID components.
func (p ParsedID) String() string {
	return "{timestamp: " + p.Timestamp.String() +
		", node: " + strconv.FormatInt(p.Node, 10) +
		", sequence: " + strconv.FormatInt(p.Sequence, 10) + "}"
}

// Format defines the bit layout of a 63-bit Snowflake ID.
//   - [0 | timestamp | node | sequence]
//   - node should be unique across data centers
type Format struct {
	TimestampBits uint8
	NodeBits      uint8
	SequenceBits  uint8
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
		return ErrFormatTooManyBits
	}
	return nil
}

type compiled struct {
	shiftTimestamp uint8
	shiftNode      uint8

	maxTimestamp int64
	maxNode      int64
	maxSeq       int64
}

func (f Format) compileFormat() compiled {
	sn := f.SequenceBits
	st := sn + f.NodeBits

	mask := func(bits uint8) int64 {
		return (int64(1) << bits) - 1
	}

	return compiled{
		shiftTimestamp: st,
		shiftNode:      sn,

		maxTimestamp: mask(f.TimestampBits),
		maxNode:      mask(f.NodeBits),
		maxSeq:       mask(f.SequenceBits),
	}
}
