package registry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
)

var errEmptyNodeIDEnv = errors.New("NODE_ID is not set in environment variables")

// StaticRegistry is a Registry that returns a fixed node ID read from the NODE_ID
// environment variable. Suitable for deployments where each instance is assigned
// a stable, pre-configured ID (e.g. via Kubernetes env injection).
type StaticRegistry struct {
	nodeID int64
}

// NewStaticRegistry creates a StaticRegistry by reading the NODE_ID environment variable.
// Returns an error if NODE_ID is unset or not a valid integer.
func NewStaticRegistry() (*StaticRegistry, error) {
	nodeIDStr := os.Getenv("NODE_ID")
	if nodeIDStr == "" {
		return nil, errEmptyNodeIDEnv
	}
	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("NODE_ID is not a valid integer: %w", err)
	}
	return &StaticRegistry{nodeID: nodeID}, nil
}

// Acquire returns the configured node ID.
func (r *StaticRegistry) Acquire(_ context.Context) (int64, error) { return r.nodeID, nil }

// Release is a no-op for static registries; the node ID is not reclaimed.
func (r *StaticRegistry) Release(_ context.Context) error { return nil }
