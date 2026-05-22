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

**Pros:**

- Throughput scales linearly with goroutine count instead of serializing
- Eliminates the 1ms latency spike where all goroutines block behind a sleeping lock holder
- No OS scheduler involvement in the hot path

**Cons:**

- CAS retry loops can spin under extreme contention (many goroutines racing simultaneously)
- More complex to implement correctly, especially for the sequence exhaustion wait case
- Sleep/Gosched during exhaustion still needs careful handling without a mutex

**When to do:** When moving to multiple goroutines per node.

---

## 2. Sub-millisecond Resolution

**Problem:** 1ms time unit means only 1024 IDs/ms/node (with default 10-bit sequence).
Sequence exhaustion is frequent under high load, causing goroutines to spin-wait.

**Idea:** Change the time unit from milliseconds to microseconds (`UnixMicro()` instead of
`UnixMilli()`). With 41-bit timestamp in microseconds: still ~34 years of range, and
sequence exhaustion happens 1000x less often.

**Pros:**

- Near-eliminates sequence exhaustion under realistic loads
- Low effort — minimal code change (swap time unit, adjust epoch range)
- No bit layout change required
- Better resolution means IDs from the same ms are now distinguishable by time

**Cons:**

- Reduces timestamp range from 69 years (ms) to ~34 years (us) with 41 bits
- IDs are no longer directly human-readable as ms timestamps without dividing by 1000
- `time.Now().UnixMicro()` has slightly higher overhead than `UnixMilli()` on some platforms

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
lifetime.

**Pros:**

- Clock backward is impossible within a process — removes an entire class of errors
- Simpler code: `ErrClockBackward`, `ErrClockSyncFailed`, drift wait, and `MaxClockDrift`
  config can all be removed
- No sleep in the hot path for clock recovery

**Cons:**

- Monotonic clock resets on process restart — IDs from different process lifetimes may not
  be wall-clock ordered relative to each other
- VM pause/resume or container migration can cause large monotonic jumps, which would appear
  as a big timestamp skip in generated IDs (not a duplicate risk, but IDs become
  non-contiguous)
- Loses the ability to decode a meaningful wall-clock timestamp from an ID without knowing
  the process start time

**When to do:** When simplifying the clock handling code is a priority.

---

## 4. Sequence Batching

**Problem:** With atomic CAS (#1), each `Generate()` still does one CAS per ID. Under very
high contention (many goroutines, many retries), CAS loops can spin.

**Idea:** A goroutine claims a batch of N sequences in one CAS (e.g., advance seq by 64),
then serves them locally from a counter with no synchronization. One CAS per N IDs.

**Pros:**

- Near-zero per-ID synchronization cost under high goroutine counts
- Reduces CAS retry rate dramatically — less contention on the shared state

**Cons:**

- Unused sequences in a batch are wasted if the ms rolls over before the batch is consumed,
  leaving gaps in the sequence space
- Adds significant implementation complexity
- Makes the sequence exhaustion boundary harder to reason about
- Premature optimization — only worth it if atomic CAS (#1) is proven insufficient

**When to do:** Only if atomic CAS alone is insufficient — this is an optimization on top
of #1, not a standalone change.

---

## Priority Order

| #   | Improvement          | Effort | Impact                       | Dependency     |
| --- | -------------------- | ------ | ---------------------------- | -------------- |
| 2   | Sub-ms resolution    | Low    | High                         | None           |
| 1   | Lock-free atomic CAS | Medium | High (multi-goroutine nodes) | None           |
| 3   | Monotonic clock      | Medium | Medium                       | None           |
| 4   | Sequence batching    | High   | Low                          | Needs #1 first |
