package redis

import "time"

// Option is a functional option for RedisRegistry.
type Option func(*redisConfig)

type redisConfig struct {
	keyPrefix          string
	ttl                time.Duration
	heartbeatInterval  time.Duration
	onHeartbeatFailure func(err error)
}

func defaultConfig() redisConfig {
	return redisConfig{
		keyPrefix:         "csid:node",
		ttl:               30 * time.Second,
		heartbeatInterval: 10 * time.Second,
	}
}

// WithKeyPrefix sets the Redis key prefix (default: "csid:node").
// Use this when multiple generator clusters share one Redis instance.
func WithKeyPrefix(prefix string) Option {
	return func(c *redisConfig) { c.keyPrefix = prefix }
}

// WithTTL sets the key TTL (default: 30s). Must be greater than 3x the heartbeat interval.
func WithTTL(d time.Duration) Option {
	return func(c *redisConfig) { c.ttl = d }
}

// WithHeartbeatInterval sets how often the node key TTL is refreshed (default: 10s).
func WithHeartbeatInterval(d time.Duration) Option {
	return func(c *redisConfig) { c.heartbeatInterval = d }
}

// WithOnHeartbeatFailure sets a callback invoked when a heartbeat refresh fails.
// Receives ErrOwnershipLost if another instance claimed the slot, or a Redis
// error for transient failures. If nil (default), transient errors are silently
// tolerated until the TTL expires.
func WithOnHeartbeatFailure(fn func(error)) Option {
	return func(c *redisConfig) { c.onHeartbeatFailure = fn }
}
