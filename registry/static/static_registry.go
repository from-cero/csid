package static

import (
	"context"
	"os"
	"strconv"

	"github.com/from-cero/csid/registry"
)

var _ registry.Registry = (*Registry)(nil)

// Registry returns a fixed node ID read from the NODE_ID environment variable.
// Suitable for deployments where each instance is assigned a stable,
// pre-configured ID (e.g. via Kubernetes env injection).
type Registry struct {
	nodeID int64
}

// NewRegistry creates a Registry by reading the NODE_ID environment variable.
// Returns an error if NODE_ID is unset or not a valid integer.
func NewRegistry() (*Registry, error) {
	nodeIDStr := os.Getenv("NODE_ID")
	if nodeIDStr == "" {
		return nil, ErrEmptyEnvNodeID
	}
	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		return nil, ErrInvalidEnvNodeID
	}
	return &Registry{nodeID: nodeID}, nil
}

// Acquire returns the configured node ID.
func (r *Registry) Acquire(_ context.Context) (int64, error) { return r.nodeID, nil }

// Release is a no-op; the node ID is not reclaimed.
func (r *Registry) Release(_ context.Context) error { return nil }
