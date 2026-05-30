package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Registry is a Registry that coordinates node ID assignment through Redis.
// Each instance atomically claims a unique slot (0...maxNodeID), holds it alive
// via TTL and a background heartbeat, and releases it on Close.
//
// If a process dies without releasing, the slot is reclaimed automatically when
// the TTL expires. All methods are safe for concurrent use.
type Registry struct {
	mu        sync.Mutex
	nodeID    int64 // -1 means not yet acquired
	acquiring bool  // true while a Redis Acquire call is in flight

	client  *goredis.Client
	cfg     config
	maxNode int64
	ownerID string             // unique per-process identity stored as the Redis value
	stopHB  context.CancelFunc // cancels the heartbeat goroutine
	hbDone  chan struct{}      // closed when the heartbeat goroutine exits
}

// NewRegistry creates a Registry. The caller provides a pre-configured
// Redis client and the inclusive upper bound of valid node IDs (e.g. 4095 for a
// 12-bit node field). The registry does not touch Redis until Acquire is called.
func NewRegistry(client *goredis.Client, maxNodeID int64, opts ...Option) (*Registry, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	if maxNodeID < 0 {
		return nil, ErrInvalidMaxNodeID
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.ttl <= 3*cfg.heartbeatInterval {
		return nil, ErrInvalidTTLConfig
	}

	ownerID, err := generateOwnerID()
	if err != nil {
		return nil, fmt.Errorf("generate owner id: %w", err)
	}

	return &Registry{
		nodeID:  -1,
		client:  client,
		cfg:     cfg,
		maxNode: maxNodeID,
		ownerID: ownerID,
	}, nil
}

// Acquire claims a free node ID slot in Redis and starts a background heartbeat
// to maintain ownership. Idempotent: a second call returns the same node ID
// without touching Redis.
func (r *Registry) Acquire(ctx context.Context) (int64, error) {
	r.mu.Lock()
	if r.nodeID != -1 {
		r.mu.Unlock()
		return r.nodeID, nil
	}
	if r.acquiring {
		r.mu.Unlock()
		return -1, ErrAcquireInProgress
	}
	r.acquiring = true
	r.mu.Unlock()

	ttlMs := r.cfg.ttl.Milliseconds()
	result, err := acquireScript.Run(
		ctx, r.client, nil,
		r.cfg.keyPrefix, strconv.FormatInt(r.maxNode, 10), r.ownerID, strconv.FormatInt(ttlMs, 10),
	).Int64()

	r.mu.Lock()
	r.acquiring = false
	if err != nil {
		r.mu.Unlock()
		return -1, err
	}
	if result == -1 {
		r.mu.Unlock()
		return -1, ErrNoNodeAvailable
	}
	r.nodeID = result
	hbCtx, cancel := context.WithCancel(context.Background())
	r.stopHB = cancel
	r.hbDone = make(chan struct{})
	r.mu.Unlock()

	go r.runHeartbeat(hbCtx, result)

	return result, nil
}

// Release stops the heartbeat goroutine, then deletes the node key in Redis.
// If the key was already gone (TTL expired), Release still returns nil.
func (r *Registry) Release(ctx context.Context) error {
	r.mu.Lock()
	id := r.nodeID
	if id == -1 {
		r.mu.Unlock()
		return ErrNotAcquired
	}
	r.nodeID = -1
	stopHB := r.stopHB
	hbDone := r.hbDone
	r.mu.Unlock()

	stopHB() // signal the heartbeat goroutine to stop
	<-hbDone // wait for heartbeat goroutine to exit before deleting the key

	key := r.nodeKey(id)
	_, err := releaseScript.Run(ctx, r.client, []string{key}, r.ownerID).Int64()
	if err != nil {
		return err
	}
	return nil
}

func (r *Registry) runHeartbeat(ctx context.Context, nodeID int64) {
	defer close(r.hbDone)

	key := r.nodeKey(nodeID)
	ttlMs := r.cfg.ttl.Milliseconds()
	ticker := time.NewTicker(r.cfg.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Bound the Redis call to half the heartbeat interval so the goroutine
			// never blocks longer than one interval regardless of client-level timeouts.
			// This upholds the guarantee that onHeartbeatFailure fires within one
			// heartbeatInterval of Redis becoming unreachable.
			callCtx, cancel := context.WithTimeout(context.Background(), r.cfg.heartbeatInterval/2)
			result, err := heartbeatScript.Run(
				callCtx, r.client, []string{key},
				r.ownerID, strconv.FormatInt(ttlMs, 10),
			).Int64()
			cancel()
			if err != nil {
				r.notifyFailure(err)
				continue // transient Redis error; keep the heartbeat alive
			}
			if result == 0 {
				// ownership was lost (key expired and was taken, or deleted externally)
				r.notifyFailure(ErrOwnershipLost)
				return
			}
		}
	}
}

// notifyFailure dispatches onHeartbeatFailure in a separate goroutine so that
// a callback calling Release does not deadlock against the heartbeat goroutine
// waiting on hbDone.
func (r *Registry) notifyFailure(err error) {
	if r.cfg.onHeartbeatFailure != nil {
		go r.cfg.onHeartbeatFailure(err)
	}
}

func (r *Registry) nodeKey(id int64) string {
	return fmt.Sprintf("%s:%d", r.cfg.keyPrefix, id)
}

func generateOwnerID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
