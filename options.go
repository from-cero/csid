package csid

import "time"

// Config holds all configuration for a Node or Parser.
type Config struct {
	Format            Format        // Default is csid.DefaultFormat.
	Epoch             time.Time     // Default is 2026-01-01 00:00:00 UTC.
	MaxClockDrift     time.Duration // Default is 10ms.
	YieldOnExhaustion bool
	// Default is false. If true, yield (runtime.Gosched) on sequence exhaustion instead of sleeping.
}

type Option func(*Config) // Option is a functional option for configuring.

// WithFormat sets the bit layout for IDs.
func WithFormat(f Format) Option { return func(c *Config) { c.Format = f } }

// WithEpoch sets the custom epoch used as the zero time for timestamps.
func WithEpoch(e time.Time) Option { return func(c *Config) { c.Epoch = e } }

// WithMaxClockDrift sets the maximum tolerated backward clock drift before.
func WithMaxClockDrift(d time.Duration) Option { return func(c *Config) { c.MaxClockDrift = d } }

// WithYieldOnExhaustion enables yielding (runtime.Gosched) instead of sleeping when the sequence
// is exhausted. This allows the node to reach its theoretical maximum throughput (~1024 IDs/ms
// with the default format) at the cost of burning a CPU core during exhaustion. Use only when
// squeezing maximum throughput from a single node; otherwise prefer multiple nodes.
func WithYieldOnExhaustion(v bool) Option { return func(c *Config) { c.YieldOnExhaustion = v } }

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
