# Redis Registry Design

## 1. Problem Statement

The Snowflake ID generator (`csid`) requires each running instance to hold a **unique node ID**
in the range `[0, maxNodeID]`. In the default 12-bit configuration that is 4096 slots.

The existing `StaticRegistry` solves this by reading a `NODE_ID` environment variable --
suitable for Kubernetes with stable pod identities. It breaks in dynamic environments:

- **Autoscaling / spot instances**: you cannot pre-assign env vars to instances that do not
  exist yet.
- **Multiple replicas with the same config**: if two pods get the same `NODE_ID`, their IDs
  will collide, silently violating the uniqueness guarantee.
- **Rolling restarts**: a pod restarting inside the same 1ms window as its predecessor could
  produce duplicate IDs (the comment in `generator.go` notes Go runtime init takes >1ms, but
  this is not guaranteed forever).

Redis provides a coordination layer where instances compete for slots at startup, hold them
via TTL, and release them on graceful shutdown.

---

## 2. Module Structure

The Redis registry lives in a **separate Go submodule**:

```
registry/redis/
  go.mod               -- module github.com/from-cero/csid/registry/redis
  errors.go
  options.go
  scripts.go
  redis_registry.go
  redis_registry_test.go
  DESIGN.md
```

**Why a separate submodule, not a file in the root?**

The root module (`github.com/from-cero/csid`) has zero external dependencies by design.
Adding `go-redis/v9` to the root `go.mod` would force every consumer to pull Redis client
code even if they only use `StaticRegistry`. A submodule makes Redis support opt-in: users
run `go get github.com/from-cero/csid/registry/redis` only when they need it. This is the
standard Go ecosystem pattern (e.g., `go.opentelemetry.io/otel` vs
`go.opentelemetry.io/otel/exporters/otlp`).

---

## 3. The Registry Interface Contract

`RedisRegistry` implements the same interface as `StaticRegistry`:

```go
type Registry interface {
    Acquire(ctx context.Context) (int64, error)
    Release(ctx context.Context) error
}
```

Go's implicit interface satisfaction means `RedisRegistry` satisfies this interface without
importing the parent module -- so the `registry/redis` submodule has **no compile-time
dependency on `github.com/from-cero/csid`**.

---

## 4. Redis Key Design

Each slot maps to one Redis key:

```
{keyPrefix}:{nodeID}
```

Default prefix is `csid:node`, so slot 42 becomes `csid:node:42`. The **value** stored at
the key is an `ownerID` -- a 16-byte random hex string generated once per `RedisRegistry`
instance at construction time:

```go
func generateOwnerID() (string, error) {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b), nil
}
```

The ownerID serves as proof of ownership. Every Lua script that mutates a key first checks
that the stored value matches the caller's ownerID before acting. This prevents one instance
from accidentally deleting or refreshing a slot it no longer owns.

---

## 5. The Three Lua Scripts

All Redis mutations use **Lua scripts** executed server-side. This is critical: Lua execution
is single-threaded and atomic on Redis -- no TOCTOU race is possible between check and mutate.

### 5.1 Acquire Script

```lua
-- ARGV[1]=keyPrefix, ARGV[2]=maxNodeID, ARGV[3]=ownerID, ARGV[4]=ttlMilliseconds
local prefix = ARGV[1]
local max    = tonumber(ARGV[2])
local owner  = ARGV[3]
local ttl    = tonumber(ARGV[4])

for i = 0, max do
    local key = prefix .. ":" .. i
    if redis.call("SET", key, owner, "NX", "PX", ttl) then
        return i
    end
end
return -1
```

**What it does:** Scans slots 0..maxNode sequentially. For each slot, attempts
`SET key ownerID NX PX ttl` -- "set only if not exists, expire after `ttl` milliseconds."
The first successful `SET` returns that slot number. If all slots are taken, returns -1.

**Why sequential scan inside Lua?**
Under a single Lua execution, each `SET NX` call is in-process (no network round-trip
between iterations). The entire scan runs in at most `maxNode` microseconds on the Redis
server. Two instances racing simultaneously both execute their own Lua script -- each `SET NX`
is individually atomic, so only one can win any given slot. No TOCTOU window exists.

**Why not a pipeline of SET NX calls?**
A pipeline issues multiple commands in one network round-trip but they execute independently --
two instances could observe the same slot free and both attempt to claim it, with only Redis
command ordering determining who wins. More importantly, the caller would need to inspect
N responses to find which slot (if any) they claimed. The Lua loop is simpler, faster per
invocation, and provably correct.

**TTL in milliseconds (`PX`)**, not seconds (`EX`): avoids integer truncation to zero for
sub-second TTLs (relevant in tests; also correct for any deployment using TTLs under 1s).

### 5.2 Heartbeat Script

```lua
-- KEYS[1]=nodeKey, ARGV[1]=ownerID, ARGV[2]=ttlMilliseconds
if redis.call("GET", KEYS[1]) == ARGV[1] then
    redis.call("PEXPIRE", KEYS[1], tonumber(ARGV[2]))
    return 1
end
return 0
```

**What it does:** Refreshes the TTL of a node key, but only if the current value matches
the caller's ownerID. Returns 1 on success, 0 if the key is missing or owned by a different
instance.

**Why check before refresh?**
If Redis evicted the key (TTL expired) and another instance claimed it, a blind `PEXPIRE`
would extend the *new* owner's TTL on behalf of the old owner -- giving the old owner an
inadvertent lease extension over a slot it no longer holds.

### 5.3 Release Script

```lua
-- KEYS[1]=nodeKey, ARGV[1]=ownerID
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
end
return 0
```

**What it does:** Deletes a node key, but only if the caller is still the owner. Returns 1
if deleted, 0 if the key is gone or owned by someone else.

This is the canonical distributed lock release pattern. Without the ownership check, a slow
instance could accidentally delete a slot that was reclaimed by a faster instance after TTL
expiry.

---

## 6. Lifecycle: Acquire, Heartbeat, Release

### 6.1 Acquire

```
1. Lock mu
2. If nodeID != -1: return nodeID (idempotent)
3. Unlock mu

4. Run acquireScript (single Redis call; respects ctx cancellation)

5. If result == -1: return ErrNoNodeAvailable
6. Lock mu; store nodeID = result; create hbCtx + hbDone; unlock

7. Start heartbeat goroutine
8. Return nodeID
```

**Why unlock before the Redis call?**
Holding the mutex during a network operation would block all callers (`Release`, a second
`Acquire`) for the full network round-trip. The mutex protects in-memory state only; the
Redis interaction is inherently serialized by the atomic Lua script.

**Context cancellation:**
`go-redis` passes the `ctx` to the underlying TCP dial/read. If cancelled mid-call, it
returns `ctx.Err()`. No partial state to clean up -- `SET NX` is all-or-nothing.

### 6.2 Heartbeat Goroutine

```
ticker := NewTicker(heartbeatInterval)
loop:
  case ctx.Done()  -> return              (Release cancelled us)
  case ticker.C    ->
    run heartbeatScript using context.Background()
    if redis error      -> call onHeartbeatFailure(err); continue
    if result == 0      -> call onHeartbeatFailure(ErrOwnershipLost); return
```

**Why `context.Background()` for Redis calls, not the heartbeat's ctx?**

The heartbeat `ctx` is cancelled only by `Release`. Its sole purpose is to detect "stop
now." The Redis calls inside the heartbeat loop must not be cancelled by an external caller
who passed a short-lived context to `Acquire` -- the heartbeat must outlive any individual
request context.

**Transient Redis errors:**
A single network hiccup should not kill the heartbeat. The goroutine calls
`onHeartbeatFailure(err)` (if set) and continues. As long as the heartbeat recovers before
the TTL expires, the slot remains held.

Each Redis call inside the heartbeat is bounded to `heartbeatInterval / 2` via a context
deadline. This ensures the goroutine never blocks longer than one interval regardless of the
client-level `DialTimeout` / `ReadTimeout` settings on the provided `*goredis.Client`,
upholding the guarantee that the callback fires within one `heartbeatInterval` of failure.

`onHeartbeatFailure` is called on **every failing tick**, not just the first. Applications
that trigger shutdown from the callback must guard against being called multiple times
(e.g., use `sync.Once`).

**Ownership lost (`result == 0`):**
The key is either gone (TTL expired and was not refreshed in time) or overwritten by another
instance. The goroutine calls `onHeartbeatFailure(ErrOwnershipLost)` and exits. The
application's callback decides policy: log and continue, trigger graceful shutdown, or
hard-exit.

### 6.3 Release

```
1. Lock mu; grab id; if id == -1: return ErrNotAcquired
2. Set nodeID = -1 (under lock); grab stopHB and hbDone
3. Unlock mu

4. stopHB()      -- signal heartbeat to stop
5. <-hbDone      -- WAIT for heartbeat to actually exit (bounded by heartbeatInterval)

6. Run releaseScript
7. Return nil (even if script returns 0 -- key was already gone)
```

**Why wait on `<-hbDone` before issuing the delete?**

Strict ordering prevents a race where the heartbeat ticker fires between `stopHB()` and the
`DEL` script: the heartbeat would call `GET` on the key we are about to delete. If a new
instance claimed the same slot in that gap (unlikely but possible at high startup rates),
the heartbeat would see the new owner's ownerID, return 0, and fire `onHeartbeatFailure` --
a false positive. Waiting ensures the heartbeat goroutine has fully exited before we touch
Redis.

**Why `result == 0` is not an error in Release?**
If the key was already gone (TTL expired naturally), the slot is freed whether we deleted it
or not. An error here would be non-actionable noise for the caller.

---

## 7. Safety Analysis: The Ownership Model

Consider two instances A and B competing for the same slot:

```
t= 0s   A acquires slot 3, stores ownerID=AAA, TTL=30s
t=25s   A heartbeat refreshes slot 3's TTL to 30s from now
t=30s   A crashes (heartbeat stops)
t=60s   Slot 3's TTL expires; Redis deletes the key
t=61s   B acquires slot 3, stores ownerID=BBB, TTL=30s

Result: A and B never hold slot 3 simultaneously. No collision.
```

**The only window where a collision is possible:**

A is generating IDs, Redis becomes unreachable, the heartbeat cannot refresh the TTL, the
TTL expires, and B claims slot 3 -- while A is still alive and generating IDs. Both A and B
hold slot 3 concurrently.

The collision window **opens** at most `TTL` after the last successful heartbeat -- that is
when the key expires and another instance can claim the slot. The `onHeartbeatFailure`
callback fires much earlier: within one `heartbeatInterval` of Redis becoming unreachable.
Because the invariant `TTL > 3 * heartbeatInterval` is enforced at construction, the
application always has at least `2 * heartbeatInterval` of margin between the first callback
and the moment any collision becomes possible.

With defaults (TTL=30s, heartbeat=10s): callback fires within 10s of failure; key expires
within 30s; the application has at least 20s to react before the slot can be claimed by
another instance.

---

## 8. Configuration Reference

| Option | Default | Notes |
|---|---|---|
| `WithKeyPrefix(string)` | `"csid:node"` | Use distinct prefixes for independent generator clusters sharing one Redis instance |
| `WithTTL(duration)` | `30s` | Must satisfy `TTL > 3 * heartbeatInterval` (enforced at construction) |
| `WithHeartbeatInterval(duration)` | `10s` | How often the TTL is refreshed |
| `WithOnHeartbeatFailure(func(error))` | `nil` | Called on transient Redis errors and on `ErrOwnershipLost`; nil means silent tolerance |

**The `TTL > 3 * heartbeatInterval` invariant** ensures the TTL is refreshed at least twice
before it expires even if one heartbeat is delayed by a slow Redis. A ratio of 3 provides
a meaningful buffer over the minimum of 2.

---

## 9. Error Reference

| Error | Returned by | Meaning |
|---|---|---|
| `ErrNoNodeAvailable` | `Acquire` | All slots 0..maxNodeID are occupied in Redis |
| `ErrNotAcquired` | `Release` | `Release` called before a successful `Acquire` |
| `ErrOwnershipLost` | `onHeartbeatFailure` callback | Heartbeat detected the key was evicted or overwritten by another instance |
| `ErrInvalidConfig` | `NewRedisRegistry` | `TTL <= 3 * heartbeatInterval` |
| `ErrInvalidMaxNodeID` | `NewRedisRegistry` | `maxNodeID < 0` |

---

## 10. Testing Approach

Tests use **`alicebob/miniredis`** -- a pure-Go in-memory implementation of the Redis
protocol. It starts in microseconds, requires no Docker, and runs in any CI environment.

**Key property exploited:** `mr.FastForward(d)` advances miniredis's virtual clock,
triggering TTL expiry instantly. This makes crash-recovery scenarios (slot reclamation after
TTL) deterministic without wall-clock waiting.

For heartbeat tests, short intervals are used (100ms heartbeat, 500ms TTL). The
`TTL > 3 * heartbeatInterval` constraint still holds (500ms > 300ms).

**Test coverage:**

| Test | What it verifies |
|---|---|
| `TestAcquire_ClaimsFirstFreeSlot` | Skips occupied slots, claims first free one |
| `TestAcquire_AllSlotsFull` | Returns `ErrNoNodeAvailable` when all slots taken |
| `TestAcquire_Idempotent` | Second call returns same ID, no extra Redis call |
| `TestRelease_DeletesKey` | Key is gone in Redis after Release |
| `TestRelease_BeforeAcquire` | Returns `ErrNotAcquired` |
| `TestRelease_WhenKeyAlreadyGone` | Returns nil even if TTL already expired the key |
| `TestHeartbeat_RefreshesTTL` | TTL stays positive after FastForward past heartbeat interval |
| `TestHeartbeat_StopsOnRelease` | Release returns within 2s (heartbeat goroutine does not block) |
| `TestHeartbeat_OwnershipLost_CallbackFired` | `onHeartbeatFailure` called with `ErrOwnershipLost` after key deleted externally |
| `TestConcurrentAcquire_NoDuplicates` | 10 goroutines acquire simultaneously, all get distinct IDs |
| `TestAcquireRelease_SlotIsReclaimable` | Slot can be re-acquired by a new instance after Release |

---

## 11. What Was Deliberately Left Out

| Feature | Reason |
|---|---|
| Redis Cluster / Sentinel | `*goredis.Client` covers the common standalone case; cluster and sentinel use distinct client types with different semantics -- a separate concern for a future version |
| Retry loop in `Acquire` | All slots full is a real operational signal; hiding it behind retries delays the signal. The caller or orchestrator decides the retry policy. |
| Sticky node IDs (same ID across restarts) | Snowflake correctness only requires node IDs to be unique *right now*, not stable across process lifetimes |
| Metrics / tracing | The `onHeartbeatFailure` callback is the integration point; the library does not impose a metrics library on the user |
| `redis.UniversalClient` interface | Over-engineering for v1; `*goredis.Client` is the right concrete type for the common case |
