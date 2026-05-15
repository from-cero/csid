package crid

import (
	"time"

	"github.com/from-cero/cero-id/registry"
)

// Config holds all parameters for constructing a Generator.
type Config struct {
	Format        Format
	DatacenterID  uint8
	IsProd        bool
	Registry      registry.Registry
	MaxClockDrift time.Duration
}

// Option is a functional option for Config.
type Option func(*Config)

func WithFormat(f Format) Option               { return func(c *Config) { c.Format = f } }
func WithDatacenter(id uint8) Option           { return func(c *Config) { c.DatacenterID = id } }
func WithProd() Option                         { return func(c *Config) { c.IsProd = true } }
func WithRegistry(r registry.Registry) Option  { return func(c *Config) { c.Registry = r } }
func WithMaxClockDrift(d time.Duration) Option { return func(c *Config) { c.MaxClockDrift = d } }

func applyOptions(opts []Option) Config {
	cfg := Config{
		Format:        DefaultFormat,
		MaxClockDrift: 5 * time.Millisecond,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}
