package ceroid

import (
	"strconv"
	"time"
)

type Config struct {
	Format   Format
	Epoch    time.Time
	MaxDrift time.Duration
}

type Option func(*Config)

func WithFormat(f Format) Option          { return func(c *Config) { c.Format = f } }
func WithEpoch(e time.Time) Option        { return func(c *Config) { c.Epoch = e } }
func WithMaxDrift(d time.Duration) Option { return func(c *Config) { c.MaxDrift = d } }

func applyOptions(opts []Option) Config {
	cfg := Config{
		Format:   DefaultFormat,
		Epoch:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		MaxDrift: 10 * time.Millisecond,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

type ID int64

func (id ID) Int64() int64 {
	return int64(id)
}

func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.String() + `"`), nil
}

type ParsedID struct {
	Timestamp time.Time
	Node      int64
	Sequence  int64
}

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
