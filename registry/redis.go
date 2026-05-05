package registry

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultTTL       = 30 * time.Second
	maxWorkerID      = 128
)

// RedisOption configures the Redis registry.
type RedisOption func(*redisRegistry)

// WithLeaseTTL sets how long a worker ID lease is held before it must be renewed.
// The heartbeat fires at TTL/2. Minimum 5 s.
func WithLeaseTTL(ttl time.Duration) RedisOption {
	return func(r *redisRegistry) {
		if ttl >= 5*time.Second {
			r.ttl = ttl
		}
	}
}

// Redis returns a Registry backed by Redis.
// The client must already be connected; the registry does not close it.
func Redis(client *redis.Client, opts ...RedisOption) Registry {
	r := &redisRegistry{
		client: client,
		ttl:    defaultTTL,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

type redisRegistry struct {
	client *redis.Client
	ttl    time.Duration
}

func (r *redisRegistry) Claim(ctx context.Context, datacenterID uint8, isProd bool) (uint8, func() error, error) {
	env := envStr(isProd)
	nodeVal := nodeIdentity()

	for id := 0; id < maxWorkerID; id++ {
		key := redisKey(datacenterID, env, id)
		ok, err := r.client.SetNX(ctx, key, nodeVal, r.ttl).Result()
		if err != nil {
			return 0, nil, fmt.Errorf("ceroflake/registry: redis SetNX: %w", err)
		}
		if !ok {
			continue
		}

		workerID := uint8(id)
		stopCh := make(chan struct{})
		go r.heartbeat(key, nodeVal, stopCh)

		release := func() error {
			close(stopCh)
			return r.client.Del(context.Background(), key).Err()
		}
		return workerID, release, nil
	}

	return 0, nil, fmt.Errorf("ceroflake/registry: %w", ErrExhausted)
}

// ErrExhausted is returned when all 128 worker slots are occupied.
var ErrExhausted = fmt.Errorf("all worker IDs are taken for this datacenter/env")

func (r *redisRegistry) heartbeat(key, nodeVal string, stop <-chan struct{}) {
	interval := r.ttl / 2
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			// Only renew if this node still owns the key.
			r.client.Eval(ctx,
				`if redis.call("GET",KEYS[1])==ARGV[1] then return redis.call("EXPIRE",KEYS[1],ARGV[2]) else return 0 end`,
				[]string{key}, nodeVal, strconv.Itoa(int(r.ttl.Seconds())),
			)
			cancel()
		}
	}
}

func redisKey(datacenterID uint8, env string, workerID int) string {
	return fmt.Sprintf("ceroflake:worker:%d:%s:%d", datacenterID, env, workerID)
}

func envStr(isProd bool) string {
	if isProd {
		return "prod"
	}
	return "nonprod"
}

func nodeIdentity() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s:%d:%d", host, os.Getpid(), rand.Int63())
}
