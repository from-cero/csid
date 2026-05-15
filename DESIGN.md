# CeroFlake — Library Design

## Overview

CeroFlake is a distributed 64-bit ID generator for Go, inspired by Twitter Snowflake.
IDs are time-ordered, embeds routing metadata (datacenter, env, worker), and carry a
domain tag (entity type) so callers never need an out-of-band table to know what a
raw ID refers to.

---

## ID Format

> **TBD — caller to provide bit layout.**

Current working layout (subject to change):

```
bit 63        0
 [0][  39 ms  ][4 entity][4 dc][1 env][7 worker][8 seq]
```

| Field      | Bits | Range     | Notes                                            |
| ---------- | ---- | --------- | ------------------------------------------------ |
| Sign       | 1    | 0         | Always 0; keeps ID positive                      |
| Timestamp  | 39   | ~17 years | Milliseconds since custom epoch (2026-05-05 UTC) |
| Entity     | 4    | 0–15      | Domain tag                                       |
| Datacenter | 4    | 0–15      | —                                                |
| Env        | 1    | 0/1       | 0 = non-prod, 1 = prod                           |
| Worker     | 7    | 0–127     | Assigned by Registry                             |
| Sequence   | 8    | 0–255     | Per-ms counter; 256 IDs/ms/worker                |

---

## Public API

### Package `ceroflake`

```go
// Construction
func New(ctx context.Context, opts ...Option) (*Generator, error)

// Core
func (g *Generator) Generate(entity EntityType) (int64, error)
func (g *Generator) Parse(id int64) ParsedID
func (g *Generator) Close() error

// Package-level parse (no Generator needed)
func Parse(id int64) ParsedID

// Options
func WithDatacenter(id uint8) Option
func WithProd() Option
func WithRegistry(r registry.Registry) Option
func WithMaxClockDrift(d time.Duration) Option
```

**Open question:** Should `Generate` return a named `ID` type instead of `int64`?
A named type prevents accidental misuse (passing a raw int where an ID is expected)
and enables attaching `Parse()`, `String()`, `MarshalText()` as methods directly on
the value. Decision blocked on final ID string format.

### Type `EntityType`

```go
type EntityType uint8

const (
    EntityGeneric  EntityType = 0
    EntityUser     EntityType = 1
    EntityOrder    EntityType = 2
    EntityProduct  EntityType = 3
    EntityPayment  EntityType = 4
    EntityInvoice  EntityType = 5
    EntityShipment EntityType = 6
    EntitySession  EntityType = 7
    // 8–15 available for caller extension
)
```

**Open question:** Should there be a way for callers to register custom entity names
(8–15) so `EntityType.String()` and `ParsedID` can render them meaningfully?

### Type `ParsedID`

```go
type ParsedID struct {
    Time       time.Time
    Entity     EntityType
    Datacenter uint8
    IsProd     bool
    WorkerID   uint8
    Sequence   uint8
}
```

### Package `registry`

```go
type Registry interface {
    Claim(ctx context.Context, datacenterID uint8, isProd bool) (workerID uint8, release func() error, err error)
}

func Static(workerID uint8) Registry   // fixed ID; for testing / single-node
func Redis(client *redis.Client, opts ...RedisOption) Registry
```

---

## Design Decisions

### Worker ID lease (Redis)

Workers scan slots 0–127 with `SETNX` and hold the key with a heartbeat (TTL/2).
On `Close()` the key is deleted immediately. This keeps coordination lock-free on
the hot path — `Generate` never touches Redis.

### Sequence exhaustion

When all 256 sequence slots are used within one millisecond, `Generate` busy-waits
for the next millisecond **while holding the mutex**. This is intentional: releasing
the lock here would let queued goroutines reuse already-issued sequence numbers.

### Clock backward

Small drift (≤ `MaxClockDrift`, default 5 ms) is absorbed by clamping `now` to
`lastMs`. Larger drift returns `ErrClockBackward` immediately rather than blocking
indefinitely.

### `Config` visibility

`Config` is currently exported but callers interact with it only through `Option`
funcs. Consider making it unexported in a future revision.

---

## Open Questions

1. **ID string format** — decimal, hex, base62, or custom? Drives `String()` /
   `MarshalText` / URL-safe representation. _Caller to provide._
2. **Named `ID` type** — `int64` vs `type ID int64` with methods.
3. **Custom entity registration** — expose `RegisterEntity(EntityType, string)` or keep it caller-managed?
4. **Batch generate** — `GenerateN(entity, n int) ([]int64, error)` for bulk use cases?
