package redis_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	csidredis "github.com/from-cero/csid/registry/redis"
)

func newTestClient(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client, mr
}

func newTestRegistry(
	t *testing.T,
	client *goredis.Client,
	maxNode int64,
	opts ...csidredis.Option,
) *csidredis.Registry {
	t.Helper()
	opts = append(
		[]csidredis.Option{
			csidredis.WithTTL(30 * time.Second),
			csidredis.WithHeartbeatInterval(9 * time.Second),
		}, opts...,
	)
	reg, err := csidredis.NewRegistry(client, maxNode, opts...)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return reg
}

func TestNewRegistry_NilClient(t *testing.T) {
	_, err := csidredis.NewRegistry(nil, 4095)
	if err == nil {
		t.Error("expected error for nil client, got nil")
	}
}

func TestNewRegistry_NegativeMaxNode(t *testing.T) {
	client, _ := newTestClient(t)
	_, err := csidredis.NewRegistry(client, -1)
	if !errors.Is(err, csidredis.ErrInvalidMaxNodeID) {
		t.Errorf("error = %v, want ErrInvalidMaxNodeID", err)
	}
}

func TestNewRegistry_InvalidConfig_TTLTooShort(t *testing.T) {
	client, _ := newTestClient(t)
	_, err := csidredis.NewRegistry(
		client, 4095,
		csidredis.WithTTL(20*time.Second),
		csidredis.WithHeartbeatInterval(10*time.Second), // 3x = 30s > 20s
	)
	if !errors.Is(err, csidredis.ErrInvalidConfig) {
		t.Errorf("error = %v, want ErrInvalidConfig", err)
	}
}

func TestAcquire_ClaimsFirstFreeSlot(t *testing.T) {
	client, mr := newTestClient(t)
	// pre-occupy slots 0 and 1
	err := mr.Set("csid:node:0", "other")
	if err != nil {
		return
	}
	err = mr.Set("csid:node:1", "other")
	if err != nil {
		return
	}

	reg := newTestRegistry(t, client, 4)
	id, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if id != 2 {
		t.Errorf("Acquire() = %d, want 2", id)
	}

	_ = reg.Release(context.Background())
}

func TestAcquire_AllSlotsFull(t *testing.T) {
	client, mr := newTestClient(t)
	for i := 0; i <= 3; i++ {
		err := mr.Set(fmt.Sprintf("csid:node:%d", i), "other")
		if err != nil {
			return
		}
	}

	reg := newTestRegistry(t, client, 3)
	_, err := reg.Acquire(context.Background())
	if !errors.Is(err, csidredis.ErrNoNodeAvailable) {
		t.Errorf("error = %v, want ErrNoNodeAvailable", err)
	}
}

func TestAcquire_Idempotent(t *testing.T) {
	client, _ := newTestClient(t)
	reg := newTestRegistry(t, client, 4)

	id1, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	id2, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("second Acquire() error = %v", err)
	}
	if id1 != id2 {
		t.Errorf("idempotent Acquire: first=%d, second=%d; want same", id1, id2)
	}

	_ = reg.Release(context.Background())
}

func TestRelease_DeletesKey(t *testing.T) {
	client, mr := newTestClient(t)
	reg := newTestRegistry(t, client, 4)

	id, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	key := fmt.Sprintf("csid:node:%d", id)
	if _, err := mr.Get(key); err != nil {
		t.Fatalf("key %s not set after Acquire: %v", key, err)
	}

	if err := reg.Release(context.Background()); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if _, err := mr.Get(key); err == nil {
		t.Errorf("key %s still exists after Release", key)
	}
}

func TestRelease_BeforeAcquire(t *testing.T) {
	client, _ := newTestClient(t)
	reg := newTestRegistry(t, client, 4)

	err := reg.Release(context.Background())
	if !errors.Is(err, csidredis.ErrNotAcquired) {
		t.Errorf("error = %v, want ErrNotAcquired", err)
	}
}

func TestRelease_WhenKeyAlreadyGone(t *testing.T) {
	client, mr := newTestClient(t)
	reg := newTestRegistry(t, client, 4)

	id, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	// externally delete the key (simulates TTL expiry)
	mr.Del(fmt.Sprintf("csid:node:%d", id))

	if err := reg.Release(context.Background()); err != nil {
		t.Errorf("Release() after key gone error = %v, want nil", err)
	}
}

func TestHeartbeat_RefreshesTTL(t *testing.T) {
	client, mr := newTestClient(t)
	// Use a short heartbeat so the ticker fires in real time within the test.
	reg, err := csidredis.NewRegistry(
		client, 4,
		csidredis.WithTTL(500*time.Millisecond),
		csidredis.WithHeartbeatInterval(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	id, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	key := fmt.Sprintf("csid:node:%d", id)

	// Record the TTL right after Acquire.
	initialTTL := mr.TTL(key)

	// Wait for at least one heartbeat to fire (2x interval for safety).
	time.Sleep(250 * time.Millisecond)

	// The heartbeat uses PEXPIRE which resets the TTL back toward the full 500ms.
	// The refreshed TTL must be >= the initial TTL (it was reset, not decremented).
	refreshedTTL := mr.TTL(key)
	if refreshedTTL < initialTTL {
		t.Errorf(
			"TTL after heartbeat = %v, want >= initial TTL %v (heartbeat did not refresh)",
			refreshedTTL,
			initialTTL,
		)
	}

	_ = reg.Release(context.Background())
}

func TestHeartbeat_StopsOnRelease(t *testing.T) {
	client, _ := newTestClient(t)
	reg := newTestRegistry(t, client, 4)

	_, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- reg.Release(context.Background())
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Release() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Release() did not return within 2s (heartbeat goroutine may be stuck)")
	}
}

func TestHeartbeat_OwnershipLost_CallbackFired(t *testing.T) {
	client, mr := newTestClient(t)

	callbackErr := make(chan error, 1)
	// Use a very short heartbeat so the goroutine fires within the test timeout.
	reg, err := csidredis.NewRegistry(
		client, 4,
		csidredis.WithTTL(500*time.Millisecond),
		csidredis.WithHeartbeatInterval(100*time.Millisecond),
		csidredis.WithOnHeartbeatFailure(
			func(err error) {
				select {
				case callbackErr <- err:
				default:
				}
			},
		),
	)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	id, acquireErr := reg.Acquire(context.Background())
	if acquireErr != nil {
		t.Fatalf("Acquire() error = %v", acquireErr)
	}
	// externally delete the key so the next heartbeat sees ownership lost
	mr.Del(fmt.Sprintf("csid:node:%d", id))

	select {
	case cbErr := <-callbackErr:
		if !errors.Is(cbErr, csidredis.ErrOwnershipLost) {
			t.Errorf("callback error = %v, want ErrOwnershipLost", cbErr)
		}
	case <-time.After(2 * time.Second):
		t.Error("onHeartbeatFailure not called within 2s after ownership lost")
	}

	// Release still runs the delete Lua script; key is already gone so result==0, but no error.
	_ = reg.Release(context.Background())
}

func TestAcquire_ConcurrentCallReturnsError(t *testing.T) {
	// Two goroutines calling Acquire simultaneously: one must return ErrAcquireInProgress
	// rather than both claiming different slots and leaking a heartbeat goroutine.
	client, _ := newTestClient(t)
	reg := newTestRegistry(t, client, 4)

	var (
		wg   sync.WaitGroup
		id1  int64
		id2  int64
		err1 error
		err2 error
	)
	wg.Add(2)
	go func() { defer wg.Done(); id1, err1 = reg.Acquire(context.Background()) }()
	go func() { defer wg.Done(); id2, err2 = reg.Acquire(context.Background()) }()
	wg.Wait()

	// Exactly one should succeed and one should get ErrAcquireInProgress.
	successID, successErr, failErr := id1, err1, err2
	if err1 != nil && err2 == nil {
		successID, successErr, failErr = id2, err2, err1
	}
	if successErr != nil {
		t.Errorf("one Acquire should succeed, got error: %v", successErr)
	}
	if !errors.Is(failErr, csidredis.ErrAcquireInProgress) {
		t.Errorf("concurrent Acquire error = %v, want ErrAcquireInProgress", failErr)
	}
	_ = successID
	_ = reg.Release(context.Background())
}

func TestHeartbeat_CallbackCanCallRelease(t *testing.T) {
	// Verifies that calling Release from inside onHeartbeatFailure does not deadlock.
	client, mr := newTestClient(t)

	releaseDone := make(chan struct{})
	// Declare reg before the closure so the closure can capture it by reference.
	var reg *csidredis.Registry
	var regErr error
	reg, regErr = csidredis.NewRegistry(
		client, 4,
		csidredis.WithTTL(500*time.Millisecond),
		csidredis.WithHeartbeatInterval(100*time.Millisecond),
		csidredis.WithOnHeartbeatFailure(
			func(cbErr error) {
				if errors.Is(cbErr, csidredis.ErrOwnershipLost) {
					// This must not deadlock.
					_ = reg.Release(context.Background())
					close(releaseDone)
				}
			},
		),
	)
	if regErr != nil {
		t.Fatalf("NewRegistry() error = %v", regErr)
	}

	id, acquireErr := reg.Acquire(context.Background())
	if acquireErr != nil {
		t.Fatalf("Acquire() error = %v", acquireErr)
	}
	mr.Del(fmt.Sprintf("csid:node:%d", id))

	select {
	case <-releaseDone:
		// success
	case <-time.After(2 * time.Second):
		t.Error("Release inside onHeartbeatFailure deadlocked or was not called within 2s")
	}
}

func TestConcurrentAcquire_NoDuplicates(t *testing.T) {
	client, _ := newTestClient(t)
	const numInstances = 10
	const maxNode = int64(9) // exactly 10 slots (0..9)

	regs := make([]*csidredis.Registry, numInstances)
	for i := range regs {
		regs[i] = newTestRegistry(t, client, maxNode)
	}

	type result struct {
		id  int64
		err error
	}
	results := make([]result, numInstances)

	var wg sync.WaitGroup
	for i := range regs {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id, err := regs[idx].Acquire(context.Background())
			results[idx] = result{id, err}
		}(i)
	}
	wg.Wait()

	seen := make(map[int64]bool)
	for i, r := range results {
		if r.err != nil {
			t.Errorf("instance %d Acquire() error = %v", i, r.err)
			continue
		}
		if seen[r.id] {
			t.Errorf("duplicate node ID %d claimed by instance %d", r.id, i)
		}
		seen[r.id] = true
	}

	for _, reg := range regs {
		_ = reg.Release(context.Background())
	}
}

func TestAcquireRelease_SlotIsReclaimable(t *testing.T) {
	client, _ := newTestClient(t)
	const maxNode = int64(0) // only one slot

	reg1 := newTestRegistry(t, client, maxNode)
	id1, err := reg1.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	if err := reg1.Release(context.Background()); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	reg2 := newTestRegistry(t, client, maxNode)
	id2, err := reg2.Acquire(context.Background())
	if err != nil {
		t.Fatalf("second Acquire() after release error = %v", err)
	}
	if id1 != id2 {
		t.Errorf("expected same slot to be reclaimed: first=%d, second=%d", id1, id2)
	}
	_ = reg2.Release(context.Background())
}
