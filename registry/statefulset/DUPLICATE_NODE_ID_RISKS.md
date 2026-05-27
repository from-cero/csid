# Risks of Duplicate Node IDs

## Ordinal Risks

Ordinal is unique by name, not by running process -- two pods with the same ordinal can briefly
coexist during:

- Stuck terminating pod + manual intervention
    - If a pod gets stuck in Terminating state (e.g. due to PVC issues or finalizers), it may remain alive for minutes.
    - During this time, if an operator manually deletes the pod (without --force), the controller may create a new pod
      immediately, leading to overlap.
- Forced deletes (--force --grace-period=0)
    - API object is removed immediately.
    - Old container/process on node may still be alive briefly.
    - StatefulSet controller may create replacement pod immediately.
    - Result: old and new pod may overlap for a short time.
- Node unreachable / network partition / split-brain
    - Especially dangerous.
    - Control plane may think node is dead and create replacement pod elsewhere.
    - Original pod may still actually run on isolated node.
    - This is classic split-brain behavior.
- Controller inconsistency / delayed reconciliation
    - If StatefulSet controller is slow to react to pod deletion or node failure, it may create replacement pod before
      old pod is fully terminated.
    - This can happen during control plane disruption or heavy load.

No built-in startup fence → new pod generates IDs before confirming old pod is dead

## Scaling and Multi-Cluster

- No cross-cluster uniqueness guarantee in multi-datacenter setup
    - Multiple clusters can independently create the same ordinal/node ID.
- Fixed bit-width caps max replicas (10 bits = 1024 max)

## Operational / Production

- Node failure → pod stuck pending up to 5min (pod-eviction-timeout) due to PVC/sticky scheduling
- Kubernetes upgrade or control plane disruption delays pod recreation
- Forced delete runbooks (common in incidents) break ordinal uniqueness assumption
- Thundering herd: all pods restart after cluster incident → hammering ID registry simultaneously
- Split-brain: network partition → two pods with same ordinal generating IDs concurrently
- Init:Error if machine ID registry is down at startup → no IDs generated, silent or loud failure
