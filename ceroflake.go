package ceroflake

import (
	"context"
	"sync"
	"time"
)

// Epoch is the custom zero-time for CeroFlake timestamps: 2026-05-05 UTC.
var Epoch = time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)

// Bit layout (63 → 0):
//
//	[1 sign=0][39 timestamp ms][4 entity][4 datacenter][1 env][7 worker][8 sequence]
const (
	bitsTimestamp  = 39
	bitsEntity     = 4
	bitsDatacenter = 4
	bitsEnv        = 1
	bitsWorker     = 7
	bitsSequence   = 8

	shiftTimestamp  = bitsEntity + bitsDatacenter + bitsEnv + bitsWorker + bitsSequence // 24
	shiftEntity     = bitsDatacenter + bitsEnv + bitsWorker + bitsSequence              // 20
	shiftDatacenter = bitsEnv + bitsWorker + bitsSequence                               // 16
	shiftEnv        = bitsWorker + bitsSequence                                         // 15
	shiftWorker     = bitsSequence                                                      // 8

	maskTimestamp  = (int64(1) << bitsTimestamp) - 1
	maskEntity     = int64(0xF)
	maskDatacenter = int64(0xF)
	maskEnv        = int64(0x1)
	maskWorker     = int64(0x7F)
	maskSequence   = int64(0xFF)

	maxSequence = (1 << bitsSequence) - 1 // 255
)

// Generator produces unique 64-bit IDs.
type Generator struct {
	mu           sync.Mutex
	datacenterID int64
	isProd       int64
	workerID     int64
	lastMs       int64
	seq          int64
	maxDrift     time.Duration
	release      func() error
}

// New creates a Generator, claiming a worker ID from the registry.
func New(ctx context.Context, opts ...Option) (*Generator, error) {
	cfg := applyOptions(opts)

	if cfg.DatacenterID > 15 {
		return nil, ErrInvalidDatacenter
	}
	if cfg.Registry == nil {
		return nil, errNoRegistry
	}

	workerID, release, err := cfg.Registry.Claim(ctx, cfg.DatacenterID, cfg.IsProd)
	if err != nil {
		return nil, err
	}
	if workerID > 127 {
		_ = release()
		return nil, ErrInvalidWorkerID
	}

	env := int64(0)
	if cfg.IsProd {
		env = 1
	}

	return &Generator{
		datacenterID: int64(cfg.DatacenterID),
		isProd:       env,
		workerID:     int64(workerID),
		maxDrift:     cfg.MaxClockDrift,
		release:      release,
	}, nil
}

// Generate produces a unique ID tagged with the given entity type.
func (g *Generator) Generate(entity EntityType) (int64, error) {
	if err := entity.Validate(); err != nil {
		return 0, err
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	now := currentMs()

	// Clamp small backward clock drift rather than releasing the lock.
	// Releasing and re-acquiring the lock while sleeping creates a window
	// where other goroutines can reuse sequence numbers, causing duplicates.
	if now < g.lastMs {
		if time.Duration(g.lastMs-now)*time.Millisecond > g.maxDrift {
			return 0, ErrClockBackward
		}
		now = g.lastMs
	}

	if now == g.lastMs {
		g.seq++
		if g.seq > maxSequence {
			// All 256 slots used this millisecond. Busy-wait for the next ms
			// while holding the lock — at most ~1 ms of spin. This is
			// intentional: releasing the lock here would let queued goroutines
			// reuse seq values 0..N that were already issued this millisecond.
			for now <= g.lastMs {
				now = currentMs()
			}
			g.seq = 0
		}
	} else {
		g.seq = 0
	}

	g.lastMs = now

	id := (now & maskTimestamp) << shiftTimestamp
	id |= int64(entity) << shiftEntity
	id |= g.datacenterID << shiftDatacenter
	id |= g.isProd << shiftEnv
	id |= g.workerID << shiftWorker
	id |= g.seq

	return id, nil
}

// ParsedID holds the decoded components of a CeroFlake ID.
type ParsedID struct {
	Time       time.Time
	Entity     EntityType
	Datacenter uint8
	IsProd     bool
	WorkerID   uint8
	Sequence   uint8
}

// Parse decodes a CeroFlake ID into its components.
func (g *Generator) Parse(id int64) ParsedID {
	return Parse(id)
}

// Parse decodes a CeroFlake ID without a Generator instance.
func Parse(id int64) ParsedID {
	ms := (id >> shiftTimestamp) & maskTimestamp
	t := Epoch.Add(time.Duration(ms) * time.Millisecond)

	return ParsedID{
		Time:       t,
		Entity:     EntityType((id >> shiftEntity) & maskEntity),
		Datacenter: uint8((id >> shiftDatacenter) & maskDatacenter),
		IsProd:     ((id >> shiftEnv) & maskEnv) == 1,
		WorkerID:   uint8((id >> shiftWorker) & maskWorker),
		Sequence:   uint8(id & maskSequence),
	}
}

// Close releases the worker ID lease back to the registry.
func (g *Generator) Close() error {
	return g.release()
}

func currentMs() int64 {
	return time.Since(Epoch).Milliseconds()
}

var errNoRegistry = errorf("ceroflake: Registry must be set; use WithRegistry()")

type constError string

func (e constError) Error() string { return string(e) }

func errorf(s string) error { return constError(s) }
