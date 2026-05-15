package registry

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 30 * time.Second

// ErrExhausted is returned when all worker slots for a datacenter/env are taken.
var ErrExhausted = errors.New("registry: all worker IDs are taken for this datacenter/env")

// RedisOption configures a Redis-backed Registry.
type RedisOption func(*redisRegistry)

// WithLeaseTTL sets the worker ID lease duration. Minimum 5 s. The heartbeat
// fires at TTL/2 to keep the lease alive.
func WithLeaseTTL(ttl time.Duration) RedisOption {
	return func(r *redisRegistry) {
		if ttl >= 5*time.Second {
			r.ttl = ttl
		}
	}
}

// Redis returns a Registry backed by Redis. The client must already be
// connected; the registry does not close it.
func Redis(client *redis.Client, opts ...RedisOption) Registry {
	r := &redisRegistry{client: client, ttl: defaultTTL}
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
	node := nodeIdentity()

	for id := 0; id < 128; id++ {
		key := redisKey(datacenterID, env, id)
		ok, err := r.client.SetNX(ctx, key, node, r.ttl).Result()
		if err != nil {
			return 0, nil, fmt.Errorf("registry: redis SetNX: %w", err)
		}
		if !ok {
			continue
		}

		stop := make(chan struct{})
		go r.heartbeat(key, node, stop)

		release := func() error {
			close(stop)
			return r.client.Del(context.Background(), key).Err()
		}
		return uint8(id), release, nil
	}

	return 0, nil, ErrExhausted
}

func (r *redisRegistry) heartbeat(key, node string, stop <-chan struct{}) {
	ticker := time.NewTicker(r.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			r.client.Eval(
				ctx,
				`if redis.call("GET",KEYS[1])==ARGV[1] then return redis.call("EXPIRE",KEYS[1],ARGV[2]) else return 0 end`,
				[]string{key},
				node,
				strconv.Itoa(int(r.ttl.Seconds())),
			)
			cancel()
		}
	}
}

func redisKey(dc uint8, env string, id int) string {
	return fmt.Sprintf("crid:worker:%d:%s:%d", dc, env, id)
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
