package statefulset

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Registry derives the node ID from the Kubernetes
// StatefulSet pod ordinal embedded in the pod hostname (e.g. "myapp-3" -> 3).
//
// Kubernetes guarantees that each pod in a StatefulSet receives a unique, stable
// ordinal, so no external coordination is required. Acquire parses the ordinal
// from the hostname once and caches it; Release clears the cache. Both methods
// are safe for concurrent use.
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
