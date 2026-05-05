package ceroflake

import (
	"time"

	"github.com/crix/ceroflake/registry"
)

// Config holds the parameters for creating a Generator.
type Config struct {
	DatacenterID     uint8
	IsProd           bool
	Registry         registry.Registry
	// MaxClockDrift is how long Generate() will block waiting for a backward
	// clock to recover before returning ErrClockBackward. Default: 5 ms.
	MaxClockDrift    time.Duration
}

// Option is a functional option for Config.
type Option func(*Config)

// WithDatacenter sets the datacenter ID (0–15).
func WithDatacenter(id uint8) Option {
	return func(c *Config) { c.DatacenterID = id }
}

// WithProd marks this node as a production node (env bit = 1).
func WithProd() Option {
	return func(c *Config) { c.IsProd = true }
}

// WithRegistry sets the worker ID registry.
func WithRegistry(r registry.Registry) Option {
	return func(c *Config) { c.Registry = r }
}

// WithMaxClockDrift sets the maximum tolerated backward clock drift.
func WithMaxClockDrift(d time.Duration) Option {
	return func(c *Config) { c.MaxClockDrift = d }
}

func applyOptions(opts []Option) Config {
	cfg := Config{MaxClockDrift: 5 * time.Millisecond}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
