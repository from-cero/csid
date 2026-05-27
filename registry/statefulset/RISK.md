# StatefulSet for Snowflake ID Generator -- Risks & Cons


## Machine ID (Ordinal) Risks

Ordinal is unique by name, not by running process -- two pods with the same ordinal can briefly
coexist during:

- Rolling updates
- Forced deletes (--force --grace-period=0)
- podManagementPolicy: Parallel
- Network partition / split-brain

No built-in startup fence -> new pod generates IDs before confirming old pod is dead

## Scaling

- Scale down -> ordinal gaps; reusing IDs after scale-up risks collision with previously issued IDs
- Scale up -> new pod must safely claim ordinal before generating, no native mechanism
- No cross-cluster uniqueness guarantee in multi-datacenter setup
- Fixed bit-width caps max replicas (10 bits = 1024 max)

## Clock Risks

- NTP step correction (backward jump) -> duplicates if not guarded
- Container clock inherits host clock -> VM live migration can cause backward jumps
- No built-in drift detection in most Snowflake implementations

## Operational / Production

- Node failure -> pod stuck pending up to 5min (pod-eviction-timeout) due to PVC/sticky scheduling
- Kubernetes upgrade or control plane disruption delays pod recreation
- Forced delete runbooks (common in incidents) break ordinal uniqueness assumption
- Thundering herd: all pods restart after cluster incident -> hammering ID registry simultaneously
- Split-brain: network partition -> two pods with same ordinal generating IDs concurrently
- Init:Error if machine ID registry is down at startup -> no IDs generated, silent or loud failure

## If No External Registry

No safe way to handle any of the above without Redis/etcd fence + heartbeat + lastTimestamp persistence