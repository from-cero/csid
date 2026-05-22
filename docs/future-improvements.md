# Future Improvements

## 1. Lock-free Atomic CAS

**Problem:** `sync.Mutex` serializes all `Generate()` calls within a node. With multiple
goroutines per node, they all queue behind the lock — one goroutine sleeps holding it and
blocks everyone else for up to 1ms during sequence exhaustion.

**Idea:** Pack `lastMs` and `seq` into a single `int64` state:

```
state = lastMs << seqBits | seq
```

Use `atomic.CompareAndSwap` to update state. If another goroutine raced ahead and the CAS
fails, read the new state and retry. No lock, no scheduler involvement.

**Impact:** Throughput scales with goroutine count instead of serializing. Eliminates the
1ms latency spike where all goroutines block behind a sleeping lock holder.

**When to do:** When moving to multiple goroutines per node.

---

## 2. Sub-millisecond Resolution

**Problem:** 1ms time unit means only 1024 IDs/ms/node (with default 10-bit sequence).
Sequence exhaustion is frequent under high load, causing goroutines to spin-wait.

**Idea:** Change the time unit from milliseconds to microseconds (`UnixMicro()` instead of
`UnixMilli()`). With 41-bit timestamp in microseconds: still ~34 years of range, and
sequence exhaustion happens 1000x less often.

**Impact:** Near-eliminates sequence exhaustion under realistic loads. No bit layout change
needed — just swap the time unit. Small adjustment to epoch range docs.

**Trade-off:** IDs are no longer directly human-readable as ms timestamps without dividing
by 1000.

**When to do:** Any time — low effort, high impact.

---

## 3. Monotonic Clock

**Problem:** Wall clock (`time.Now().UnixMilli()`) can go backward (NTP slew, leap seconds,
VM migration). The current code handles this with drift detection, waiting, and multiple
error types (`ErrClockBackward`, `ErrClockSyncFailed`).

**Idea:** Use the monotonic component of `time.Now()` measured against a fixed start point:

```go
// at Node creation:
start     = time.Now()
startUnix = start.UnixMilli()

// in nowMs():
return startUnix + time.Since(start).Milliseconds()
```

`time.Since` uses the monotonic clock internally — it never goes backward within a process
lifetime. Clock backward handling code can be removed entirely.

**Impact:** Simpler code, no drift errors, no sleep in the hot path.

**Trade-off:** Monotonic clock resets on process restart, so IDs from different process
lifetimes may not be wall-clock ordered relative to each other. Acceptable for most use
cases.

**When to do:** When simplifying the clock handling code is a priority.

---

## 4. Sequence Batching

**Problem:** With atomic CAS (#1), each `Generate()` still does one CAS per ID. Under very
high contention (many goroutines, many retries), CAS loops can spin.

**Idea:** A goroutine claims a batch of N sequences in one CAS (e.g., advance seq by 64),
then serves them locally from a counter with no synchronization. One CAS per N IDs.

**Impact:** Near-zero contention overhead per ID under high goroutine counts.

**Trade-off:** Adds complexity. Unused sequences in a batch are wasted if the ms rolls over
before the batch is consumed.

**When to do:** Only if atomic CAS alone is insufficient — this is an optimization on top
of #1, not a standalone change.

---

## Priority Order

| # | Improvement | Effort | Impact | Dependency |
|---|---|---|---|---|
| 2 | Sub-ms resolution | Low | High | None |
| 1 | Lock-free atomic CAS | Medium | High (multi-goroutine nodes) | None |
| 3 | Monotonic clock | Medium | Medium | None |
| 4 | Sequence batching | High | Low | Needs #1 first |
