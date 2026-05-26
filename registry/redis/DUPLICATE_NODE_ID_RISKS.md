# Duplicate Node ID: Risks and Gaps

This document tracks what the Redis registry solves, what it does not solve, and under
what exact conditions a duplicate node ID -- and therefore a duplicate generated ID -- can
occur.

---

## What Is Solved

### 1. Concurrent startup race
**Risk:** Two instances start simultaneously and both claim the same slot.

**How solved:** The acquire Lua script runs atomically on Redis. `SET key owner NX PX ttl`
is all-or-nothing per slot. Lua execution is single-threaded on Redis, so two scripts
running concurrently cannot both succeed on the same slot. One wins, the other advances to
the next slot.

**Status: Eliminated.**

---

### 2. Crash without graceful release
**Risk:** An instance crashes (OOM, kill -9, hardware failure) without calling `Release`.
The slot is held in Redis forever, reducing available capacity.

**How solved:** Every slot key has a TTL. If the heartbeat stops (because the process is
dead), the key expires and the slot becomes available automatically.

**Status: Eliminated within one TTL window (default 30s).**

---

### 3. Stale slot from a previous process lifetime
**Risk:** A process restarts. The previous lifetime's slot key may still be in Redis with
the old ownerID. The new process must not accidentally inherit or conflict with it.

**How solved:** Each `NewRedisRegistry` call generates a new `ownerID` (`crypto/rand`
UUID). The new instance's Lua script scans with `SET NX` -- it will skip any key that still
exists from the previous lifetime. If the previous key expired, the new instance can claim
that slot freely.

**Status: Eliminated.**

---

### 4. Accidental deletion of another instance's slot during release
**Risk:** Instance A's TTL expires. Instance B claims slot 3. Then A's `Release` runs late
and deletes slot 3, which now belongs to B.

**How solved:** The release Lua script checks `GET key == ownerID` before `DEL`. A's
`ownerID` does not match B's, so the `DEL` is skipped. B keeps its slot.

**Status: Eliminated.**

---

### 5. Heartbeat refreshing the wrong instance's key
**Risk:** Same scenario -- A's TTL expires, B claims slot 3, A's heartbeat tries to refresh.

**How solved:** The heartbeat Lua script checks `GET key == ownerID` before `PEXPIRE`. A
sees a mismatch, returns 0, fires `onHeartbeatFailure(ErrOwnershipLost)`, and the heartbeat
goroutine exits.

**Status: Eliminated.**

---

### 6. Concurrent Acquire calls on the same registry leaking a heartbeat goroutine
**Risk:** Two goroutines call `Acquire` on the same `RedisRegistry` simultaneously. Both
see `nodeID == -1`, both run the Lua script, both claim different slots. Only the second
slot is tracked; the first heartbeat goroutine runs forever with no cancel handle.

**How solved:** The `acquiring bool` flag under the mutex makes concurrent Acquire calls
return `ErrAcquireInProgress` after the first call wins the flag. Only one Redis call is
ever in flight per registry instance.

**Status: Eliminated.**

---

### 7. Heartbeat Redis call blocking past heartbeat interval
**Risk:** If the user's `*goredis.Client` has long `DialTimeout` / `ReadTimeout` settings,
a Redis failure could block the heartbeat goroutine for minutes, delaying
`onHeartbeatFailure` well past the `heartbeatInterval` boundary and into the TTL expiry
window.

**How solved:** Each heartbeat Redis call is wrapped in
`context.WithTimeout(ctx, heartbeatInterval/2)`. The call is always bounded to half the
tick interval, regardless of client-level timeout settings.

**Status: Eliminated.**

---

### 8. Calling Release from inside onHeartbeatFailure deadlocking
**Risk:** The natural response to `ErrOwnershipLost` is to call `Release` in the callback.
`Release` blocks on `<-hbDone`, which is closed by the heartbeat goroutine's `defer` --
but the goroutine is blocked waiting for the callback to return. Deadlock.

**How solved:** `onHeartbeatFailure` is always dispatched via `go r.cfg.onHeartbeatFailure(err)`.
The heartbeat goroutine never blocks on the callback.

**Status: Eliminated.**

---

## What Is NOT Solved -- Remaining Gaps

### Gap 1: Redis downtime after a successful Acquire (Primary residual risk)

**Scenario:**
```
T= 0s   Instance A holds slot 3. Last heartbeat just refreshed TTL to 30s.
T= 1s   Redis becomes unreachable.
T=10s   First heartbeat attempt fails. onHeartbeatFailure(err) fires.
T=30s   Slot 3's TTL expires. Redis deletes the key.
T=31s   Instance B starts, claims slot 3.
         A and B are now both generating IDs with node ID 3 -> COLLISION.
```

**What the implementation gives you:**
The `onHeartbeatFailure` callback fires within one `heartbeatInterval` of Redis going down
(10s with defaults). The TTL > 3x heartbeat invariant guarantees at least `2 * heartbeatInterval`
of margin (20s with defaults) between the first callback and the moment any collision
becomes possible.

**What it does NOT do:**
It does not shut A down automatically. The application owns the policy via the callback.
If the callback does nothing (default `nil`), A keeps generating IDs and a collision will
occur after TTL expiry.

**Mitigation:** Always set `WithOnHeartbeatFailure` in production. The minimum safe
response is to stop the Node from generating new IDs (call `node.Close`) when the callback
receives a non-transient error (i.e., after N consecutive failures, or immediately on
`ErrOwnershipLost`).

**Status: Application responsibility. Not automatically safe with nil callback.**

---

### Gap 2: Process pause longer than TTL (hard to defend against)

**Scenario:**
```
T= 0s   Instance A holds slot 3 (TTL=30s). Last heartbeat at T=0.
T= 1s   A is paused (container cgroup freeze, VM live migration, long GC stop-the-world).
T=31s   Slot 3's key expires.
T=32s   Instance B claims slot 3.
T=35s   A resumes. Its heartbeat fires, sees ErrOwnershipLost, callback notifies.
         But between T=32s and A's shutdown: A generates IDs with node ID 3.
         B also holds node ID 3 -> COLLISION window.
```

**Why the implementation cannot prevent this:**
The heartbeat goroutine is a Go goroutine. If the entire OS process is paused, no goroutines
run. The TTL keeps counting down on Redis. When the process resumes, it will detect the
problem on the next heartbeat -- but it was already generating IDs while paused, and another
instance may have claimed the slot in the gap.

**Mitigation options:**
- Keep TTL large relative to expected pause durations.
- Use monotonic process liveness signals (e.g., Kubernetes liveness probes + eviction) to
  prevent VM migration while the process is active.
- Accept the risk if the paused-process scenario is operationally impossible in your
  environment.

**Status: Not solvable within the current design. Requires operational controls.**

---

### Gap 3: Redis restart with data loss

**Scenario:** Redis restarts with persistence disabled (or before the latest snapshot syncs).
All slot keys are gone. Every running instance thinks it still owns its slot (no heartbeat
failure yet). All instances are now generating IDs. A new instance starting immediately
after Redis restarts claims slot 0. The existing slot 0 holder is still running.

**Window of exposure:** At most one `heartbeatInterval` -- the next heartbeat tick for each
running instance will call the Lua script, get `result == 0` (key gone), fire
`ErrOwnershipLost`, and the application can shut the instance down. But during that one
interval, all running instances and any newly started instances are operating with
potentially overlapping node IDs.

**Mitigation:**
- Enable Redis AOF persistence with `appendfsync everysec` or `always`.
- Use Redis Sentinel or a replicated setup so a restart does not imply data loss.
- The current implementation does not enforce or check Redis persistence settings.

**Status: Not solvable in this library. Requires Redis configuration / infrastructure.**

---

### Gap 4: No fencing on the ID generator after ownership loss is detected

**Scenario:**
1. Heartbeat detects `ErrOwnershipLost`, calls `go onHeartbeatFailure(ErrOwnershipLost)`.
2. The callback calls `node.Close()` (or `reg.Release()`).
3. But `node.Close()` acquires a mutex. If a `Generate()` call is in flight (e.g., blocked
   on sequence exhaustion `time.Sleep`), `Close()` blocks until `Generate()` returns.
4. During this window, the generator is still producing IDs with the lost node ID.

**How large is this window?**
`Generate()` sleeps at most 1ms (sequence exhaustion) or `MaxClockDrift` (clock backward
tolerance, default 10ms). So the window is at most ~10ms after ownership loss is detected.

**In practice:** Given the callback fires asynchronously (via `go`), and `node.Close()` is
called from the callback, the real window is:
`heartbeatInterval` (until detection) + goroutine scheduling latency + up to 10ms for
an in-flight `Generate()` to complete.

**Status: Inherent to the mutex-based generator design. Acceptable in practice (<10ms).**

---

### Gap 5: No retry on ErrNoNodeAvailable

If all `maxNodeID + 1` slots are occupied at the moment of `Acquire`, the call returns
`ErrNoNodeAvailable` immediately with no retry. If instances are rolling (shutting down and
starting up at the same time), a slot may become available within seconds but the new
instance fails to start.

**Status: By design. The caller or orchestration layer owns retry policy.**

---

## Summary Table

| Risk | Solved? | Mechanism |
|---|---|---|
| Concurrent startup race | Yes | Atomic Lua SET NX |
| Crash without release | Yes | Key TTL auto-expiry |
| Stale slot from prior lifetime | Yes | New ownerID per process |
| Late Release deleting wrong owner's key | Yes | Ownership check Lua in release |
| Heartbeat refreshing wrong owner's key | Yes | Ownership check Lua in heartbeat |
| Concurrent Acquire goroutine leak | Yes | `acquiring` flag |
| Heartbeat blocked past interval | Yes | `heartbeatInterval/2` call timeout |
| Deadlock calling Release from callback | Yes | Callback dispatched via goroutine |
| Redis downtime after Acquire | **Partial** | Callback fires in time; app must act |
| Process pause longer than TTL | **No** | Operational controls required |
| Redis restart with data loss | **No** | Redis persistence config required |
| Generator fence after ownership loss | **Partial** | <10ms window; inherent to design |
| No slot available at startup | **By design** | Caller owns retry policy |
