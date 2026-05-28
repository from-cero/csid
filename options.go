package csid

import "time"

// Option is a functional option for configuring.
type Option func(*config)

// WithFormat sets the bit layout for IDs.
func WithFormat(fOpts ...FormatOption) Option {
	return func(c *config) { c.format = applyFormatOptions(fOpts) }
}

// WithEpoch sets the custom epoch used as the zero time for timestamps.
func WithEpoch(e time.Time) Option { return func(c *config) { c.epoch = e } }

// WithMaxClockDrift sets the maximum tolerated backward clock drift before.
func WithMaxClockDrift(d time.Duration) Option { return func(c *config) { c.maxClockDrift = d } }

// WithYieldOnExhaustion enables yielding (runtime.Gosched) instead of sleeping when the sequence
// is exhausted. This allows the node to reach its theoretical maximum throughput (~1024 IDs/ms
// with the default format) at the cost of burning a CPU core during exhaustion. Use only when
// squeezing maximum throughput from a single node; otherwise prefer multiple nodes.
func WithYieldOnExhaustion(v bool) Option { return func(c *config) { c.yieldOnExhaustion = v } }

type config struct {
	format            format        // Default is csid.defaultFormat.
	epoch             time.Time     // Default is 2026-01-01 00:00:00 UTC.
	maxClockDrift     time.Duration // Default is 10ms.
	yieldOnExhaustion bool          // Default is false. If true, yield (runtime.Gosched) on sequence exhaustion instead of sleeping.
}

var defaultConfig = config{
	format:            defaultFormat,
	epoch:             time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	maxClockDrift:     10 * time.Millisecond,
	yieldOnExhaustion: false,
}

func applyOptions(opts []Option) config {
	cfg := defaultConfig
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
