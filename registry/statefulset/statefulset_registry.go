package statefulset

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Registry derives the node ID from the Kubernetes StatefulSet pod ordinal
// embedded in the pod hostname (e.g. "myapp-3" -> 3).
//
// Kubernetes guarantees that each pod in a StatefulSet receives a unique, stable
// ordinal, so no external coordination is required for the common case. Acquire
// parses the ordinal from the hostname once and caches it; Release clears the
// cache. Both methods are safe for concurrent use.
//
// # Production setup
//
// For safe production use, apply all the following in your StatefulSet manifest:
//
//  1. Set updateStrategy.rollingUpdate.maxUnavailable: 1 (the default) and
//     never perform forced deletes (kubectl delete pod --force --grace-period=0)
//     unless you accept a brief duplicate-ID window. Document this in your
//     runbooks.
//
//  2. Add a preStop sleep >= your application's shutdown flush time so the old
//     pod stops generating IDs before the new one starts:
//
//     lifecycle:
//     preStop:
//     exec:
//     command: ["sh", "-c", "sleep 5"]
//
//  3. Set terminationGracePeriodSeconds to at least preStop sleep + 10s.
//
//  4. Avoid scaling down and then reusing the same ordinal range for a new
//     workload. Ordinals that previously issued IDs should be considered
//     "burned" until the ID TTL or retention window has passed.
//
// # Unresolvable risks (WARNING)
//
// The following risks CANNOT be fully eliminated without an external registry
// (Redis, etcd, or similar):
//
//   - Rolling update overlap: K8s terminates the old pod and starts the new one
//     with the same ordinal. During terminationGracePeriodSeconds both processes
//     may be alive. preStop sleep reduces the window but does not close it.
//     There is no built-in fence.
//
//   - Forced delete (--force --grace-period=0): K8s recreates the pod
//     immediately. The old process may still be running on the node. Two
//     processes will share the same node ID with no coordination.
//
//   - Clock drift on restart: lastMs is in-memory only and resets to 0 on every
//     New(). If the wall clock has drifted backward since the last run, the
//     generator will reissue timestamps it already used. The generator's
//     MaxClockDrift guard only covers live drift, not cross-restart drift.
//
//   - Split-brain / network partition: two pods with the same ordinal can operate
//     independently if the network splits after startup.
//
// If your workload cannot tolerate any duplicate IDs, use the Redis registry
// instead. See DUPLICATE_NODE_ID_RISKS.md for the full risk breakdown.
type Registry struct {
	cfg     config
	maxNode int64

	mu     sync.Mutex
	nodeID int64 // -1 means not yet acquired
}

// NewRegistry creates a Registry. maxNodeID is the inclusive
// upper bound of valid node IDs (e.g. 4095 for a 12-bit node field). The registry
// does not read the hostname until Acquire is called.
func NewRegistry(maxNodeID int64, opts ...Option) (*Registry, error) {
	if maxNodeID < 0 {
		return nil, ErrInvalidMaxNodeID
	}
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return &Registry{
		cfg:     cfg,
		maxNode: maxNodeID,
		nodeID:  -1,
	}, nil
}

// Acquire resolves the pod name, parses the StatefulSet ordinal from its last
// dash-separated segment, validates it against maxNodeID, and returns it.
// Idempotent: subsequent calls return the cached ordinal without re-reading the
// hostname.
func (r *Registry) Acquire(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.nodeID != -1 {
		return r.nodeID, nil
	}

	podName, err := r.cfg.podNameFn()
	if err != nil {
		return -1, fmt.Errorf("resolve pod name: %w", err)
	}

	ordinal, err := parseOrdinal(podName)
	if err != nil {
		return -1, err
	}

	if ordinal > r.maxNode {
		return -1, fmt.Errorf("%w: ordinal %d exceeds maxNodeID %d", ErrOrdinalOutOfRange, ordinal, r.maxNode)
	}

	r.nodeID = ordinal
	return ordinal, nil
}

// Release clears the cached node ID. It is a no-op with respect to Kubernetes;
// the ordinal is managed by the StatefulSet controller, not by this library.
func (r *Registry) Release(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.nodeID == -1 {
		return ErrNotAcquired
	}
	r.nodeID = -1
	return nil
}

// parseOrdinal extracts the non-negative integer after the last '-' in podName.
// StatefulSet pod names follow the pattern "<statefulset-name>-<ordinal>".
func parseOrdinal(podName string) (int64, error) {
	idx := strings.LastIndex(podName, "-")
	if idx == -1 || idx == len(podName)-1 {
		return -1, fmt.Errorf("%w: %q", ErrInvalidHostname, podName)
	}
	ordinal, err := strconv.ParseInt(podName[idx+1:], 10, 64)
	if err != nil || ordinal < 0 {
		return -1, fmt.Errorf("%w: %q", ErrInvalidHostname, podName)
	}
	return ordinal, nil
}
