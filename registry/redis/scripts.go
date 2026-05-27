package redis

import goredis "github.com/redis/go-redis/v9"

// acquireScript atomically scans slots 0...maxNode and claims the first free one.
// ARGV[1]=keyPrefix, ARGV[2]=maxNodeID, ARGV[3]=ownerID, ARGV[4]=ttlMilliseconds
// Returns the claimed node ID as an integer, or -1 if all slots are occupied.
var acquireScript = goredis.NewScript(
	`
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
`,
)

// heartbeatScript refreshes the TTL of a node key only if the current value
// matches the expected ownerID. Returns 1 if refreshed, 0 if key is missing
// or owned by a different instance (ownership lost).
// KEYS[1]=nodeKey, ARGV[1]=ownerID, ARGV[2]=ttlMilliseconds
var heartbeatScript = goredis.NewScript(
	`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    redis.call("PEXPIRE", KEYS[1], tonumber(ARGV[2]))
    return 1
end
return 0
`,
)

// releaseScript deletes a node key only if the current value matches the
// expected ownerID. Returns 1 if deleted, 0 if key is missing or owned by
// a different instance.
// KEYS[1]=nodeKey, ARGV[1]=ownerID
var releaseScript = goredis.NewScript(
	`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
end
return 0
`,
)
