# Custom Snowflake-style ID Generator for Go

A 63-bit Snowflake-style distributed ID generator for Go.

`csid` produces time-sortable, unique 64-bit IDs (stored in a signed `int64`, so 63 usable bits) across many nodes
without central coordination at generation time. Each ID encodes a timestamp, a node ID, and a per-millisecond sequence
number. Node IDs are handed out by a pluggable `Registry`, so you can choose how uniqueness is guaranteed for your
deployment.

## Features

- 63-bit IDs that fit in a signed `int64` and sort by creation time.
- Configurable bit layout (timestamp / node / sequence) that must sum to 63.
- Pluggable node ID allocation via the `Registry` interface.
- Three registry implementations: static (env var), Kubernetes StatefulSet ordinal, and Redis.
- Safe JSON marshaling as a quoted string to avoid precision loss in JavaScript.
- Clock-backward detection with configurable drift tolerance.
- Standalone `Parser` to decode IDs without a running node.

## Installation

```sh
go get -u github.com/from-cero/csid
```

The Redis and StatefulSet registries are separate Go modules so the core library stays dependency-free:

```sh
go get -u github.com/from-cero/csid/registry/redis
go get -u github.com/from-cero/csid/registry/statefulset
```

## ID layout

The default format allocates the 63 bits as follows:

| Field     | Bits | Range                     |
|-----------|------|---------------------------|
| Timestamp | 41   | ~69 years of milliseconds |
| Node      | 12   | 4096 nodes                |
| Sequence  | 10   | 1024 IDs per ms per node  |

The layout is configurable as long as the three fields sum to 63 (see [Custom format](#custom-format)).

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/from-cero/csid"
	"github.com/from-cero/csid/registry/static"
)

func main() {
	ctx := context.Background()

	// Read the node ID from the NODE_ID environment variable.
	reg, err := static.NewRegistry("NODE_ID")
	if err != nil {
		panic(err)
	}

	node, err := csid.New(ctx, reg)
	if err != nil {
		panic(err)
	}
	defer node.Close(ctx)

	id, err := node.Generate()
	if err != nil {
		panic(err)
	}

	fmt.Println(id.String()) // decimal string
	fmt.Println(id.Int64())  // raw int64
}
```

## Configuration

`csid.New` accepts functional options:

| Option                        | Default        | Description                                                             |
|-------------------------------|----------------|-------------------------------------------------------------------------|
| `WithFormat(...)`             | 41 / 12 / 10   | Bit layout for timestamp, node, and sequence.                           |
| `WithEpoch(time.Time)`        | 2026-01-01 UTC | Zero time for timestamps. Older epochs leave less room before overflow. |
| `WithMaxClockDrift(d)`        | 10ms           | Backward clock drift tolerated by waiting before an error is returned.  |
| `WithYieldOnExhaustion(bool)` | false          | Yield instead of sleeping on sequence exhaustion for max throughput.    |

### Custom format

```go
node, err := csid.New(ctx, reg,
    csid.WithFormat(
        csid.WithTimestampBits(42),
        csid.WithNodeBits(11),
        csid.WithSequenceBits(10),
    ),
)
```

The bits must sum to 63 or `New` returns `ErrInvalidBitFormat`.

## Parsing IDs

Use a `Parser` to decode an ID back into its components without a running node. Configure it with the same options used
to generate the IDs:

```go
parser, err := csid.NewParser()
if err != nil {
    panic(err)
}

parsed := parser.Parse(id)
fmt.Println(parsed.Timestamp) // time.Time
fmt.Println(parsed.Node) // int64
fmt.Println(parsed.Sequence) // int64
```

## Registries

A `Registry` allocates a unique node ID for each generator instance:

```go
type Registry interface {
    Acquire(ctx context.Context) (nodeID int64, err error)
    Release(ctx context.Context) error
}
```

Pick the implementation that matches how your infrastructure guarantees node uniqueness.

### Static

Reads a fixed node ID from an environment variable (defaults to `NODE_ID` when the key is empty). Best when each
instance is assigned a stable, pre-configured ID. `Release` is a no-op.

```go
reg, err := static.NewRegistry("NODE_ID")
```

### StatefulSet

Derives the node ID from a Kubernetes StatefulSet pod ordinal embedded in the hostname (e.g. `myapp-3` -> `3`).

```go
import "github.com/from-cero/csid/registry/statefulset"

reg := statefulset.NewRegistry()
```

> **Warning:** Ordinal uniqueness is by pod name, not by running process. Forced deletes, stuck-terminating pods, and
> network partitions can briefly run two pods with the same ordinal. See
`registry/statefulset/DUPLICATE_NODE_ID_RISKS.md`
> and the package docs for required manifest settings (preStop sleep, termination grace period). If you cannot tolerate
> any duplicate IDs, use the Redis registry instead.

### Redis

Coordinates node ID assignment through Redis. Each instance atomically claims a free slot (`0...maxNodeID`), keeps it
alive with a TTL and a background heartbeat, and releases it on `Close`. If a process dies without releasing, the slot
is reclaimed when the TTL expires.

```go
import (
"github.com/from-cero/csid/registry/redis"
goredis "github.com/redis/go-redis/v9"
)

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})

// maxNodeID is the inclusive upper bound of valid node IDs
// (e.g. 4095 for the default 12-bit node field).
reg, err := redis.NewRegistry(client, 4095,
redis.WithKeyPrefix("csid:node"),
redis.WithTTL(30*time.Second),
redis.WithHeartbeatInterval(10*time.Second),
redis.WithOnHeartbeatFailure(func (err error) {
// react to lost ownership or transient Redis errors
}),
)
```

The TTL must be greater than three times the heartbeat interval.

## Concurrency and clock behavior

- `Node.Generate` is safe for concurrent use; all generation is serialized through a mutex.
- If the clock moves backward by less than `maxClockDrift`, the generator waits for it to catch up; beyond that it
  returns `ErrClockBackward`.
- If the per-millisecond sequence is exhausted, the generator waits for the next millisecond (or yields when
  `WithYieldOnExhaustion(true)` is set).
- After `Close`, any call to `Generate` returns `ErrNodeClosed`.

## Development

```sh
make format        # run the formatting toolchain
make lint          # golangci-lint run ./...
make test          # go test -race -cover ./...
make test-coverage # generate and open an HTML coverage report
make precommit     # format + lint + test
```

Install the tooling with `make format-tools` and `make lint-tools`.
