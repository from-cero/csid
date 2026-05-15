package crid

import (
	"context"
	"sync"
	"time"

	"github.com/from-cero/cero-id/registry"
)

// Epoch is the custom zero-time for Cero ID timestamps: 2026-05-05 UTC.
var Epoch = time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)

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

// Generator produces unique 64-bit Snowflake IDs.
type Generator struct {
	mu      sync.Mutex
	dc      int64
	env     int64
	worker  int64
	lastMs  int64
	seq     int64
	drift   time.Duration
	release func() error
	c       compiled
}

// New creates a Generator, claiming a worker ID from the registry.
func New(ctx context.Context, opts ...Option) (*Generator, error) {
	cfg := applyOptions(opts)

	if err := cfg.Format.Validate(); err != nil {
		return nil, err
	}
	c := compileFormat(cfg.Format)

	if int64(cfg.DatacenterID) > c.maxDC {
		return nil, ErrInvalidDatacenter
	}
	if cfg.Registry == nil {
		return nil, ErrNoRegistry
	}

	workerID, release, err := cfg.Registry.Claim(ctx, cfg.DatacenterID, cfg.IsProd)
	if err != nil {
		return nil, err
	}
	if int64(workerID) > c.maxWorker {
		_ = release()
		return nil, ErrInvalidWorkerID
	}

	env := int64(0)
	if cfg.IsProd && cfg.Format.EnvBits > 0 {
		env = 1
	}

	return &Generator{
		dc:      int64(cfg.DatacenterID),
		env:     env,
		worker:  int64(workerID),
		drift:   cfg.MaxClockDrift,
		release: release,
		c:       c,
	}, nil
}

// Generate produces a unique ID tagged with the given entity type.
func (g *Generator) Generate(entity EntityType) (int64, error) {
	if int64(entity) > g.c.maxEntity {
		return 0, ErrInvalidEntity
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	now := nowMs()

	if now < g.lastMs {
		if time.Duration(g.lastMs-now)*time.Millisecond > g.drift {
			return 0, ErrClockBackward
		}
		now = g.lastMs
	}

	if now == g.lastMs {
		g.seq++
		if g.seq > g.c.maxSeq {
			for now <= g.lastMs {
				now = nowMs()
			}
			g.seq = 0
		}
	} else {
		g.seq = 0
	}

	g.lastMs = now

	id := (now & g.c.maskTimestamp) << g.c.shiftTimestamp
	id |= int64(entity) << g.c.shiftEntity
	id |= g.dc << g.c.shiftDatacenter
	id |= g.env << g.c.shiftEnv
	id |= g.worker << g.c.shiftWorker
	id |= g.seq

	return id, nil
}

// Parse decodes id using this generator's format.
func (g *Generator) Parse(id int64) ParsedID {
	return parseWith(id, g.c)
}

// Close releases the worker ID lease back to the registry.
func (g *Generator) Close() error {
	return g.release()
}

func nowMs() int64 {
	return time.Since(Epoch).Milliseconds()
}
