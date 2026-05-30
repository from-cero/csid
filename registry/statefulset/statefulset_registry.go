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
// Ordinal uniqueness is by name, not by running process. Two pods can share the
// same ordinal briefly in several scenarios (see below). Acquire parses the
// ordinal from the hostname once and caches it; Release clears the cache. Both
// methods are safe for concurrent use.
//
// # Production setup
//
// For safe production use, apply all the following in your StatefulSet manifest:
//
//  1. Never perform forced deletes (kubectl delete pod --force --grace-period=0)
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
//  4. Avoid reusing the same ordinal range after scaling down. Ordinals that
//     previously issued IDs should be considered burned until the ID TTL or
//     retention window has passed.
//
// # Risks (WARNING)
//
// Ordinal uniqueness is NOT guaranteed. The following scenarios can cause two
// pods to share the same ordinal, and therefore the same node ID, with no
// built-in fence:
//
//   - Stuck terminating pod: a pod stuck in Terminating state (e.g. due to PVC
//     issues or finalizers) keeps running. If an operator then deletes it without
//     --force, the controller may immediately create a replacement with the same
//     ordinal, causing overlap.
//
//   - Forced delete (--force --grace-period=0): the API object is removed
//     immediately and the controller creates a replacement right away. The old
//     container on the node may still be alive, so both processes run with the
//     same node ID.
//
//   - Node unreachable / network partition / split-brain: the control plane may
//     declare the node dead and schedule a replacement pod elsewhere while the
//     original pod is still running on the isolated node. Both pods generate IDs
//     concurrently with no coordination.
//
//   - Controller inconsistency / delayed reconciliation: during control plane
//     disruption or heavy load the StatefulSet controller may create a
//     replacement before the old pod is fully terminated.
//
//   - No cross-cluster uniqueness: multiple clusters independently assign the
//     same ordinals, so node IDs are not unique across clusters in a
//     multi-datacenter setup.
//
// If your workload cannot tolerate any duplicate IDs, use the Redis registry
// instead. See DUPLICATE_NODE_ID_RISKS.md for the full risk breakdown.
type Registry struct {
	mu     sync.Mutex
	nodeID int64 // -1 means not yet acquired

	cfg config
}

// NewRegistry creates a Registry. The registry does not read the hostname
// until Acquire is called.
func NewRegistry(opts ...Option) *Registry {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Registry{
		nodeID: -1,
		cfg:    cfg,
	}
}

// Acquire resolves the pod name, parses the StatefulSet ordinal from its last
// dash-separated segment, and returns it. Idempotent: subsequent calls return
// the cached ordinal without re-reading the hostname.
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
		return -1, fmt.Errorf("%w: %q", ErrInvalidPodName, podName)
	}
	ordinal, err := strconv.ParseInt(podName[idx+1:], 10, 64)
	if err != nil || ordinal < 0 {
		return -1, fmt.Errorf("%w: %q", ErrInvalidPodName, podName)
	}
	return ordinal, nil
}
