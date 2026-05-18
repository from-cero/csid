package registry

import (
	"context"
	"fmt"
	"os"
	"strconv"
)

type StaticRegistry struct {
	nodeID int64
}

func NewStaticRegistry() (*StaticRegistry, error) {
	nodeIDStr := os.Getenv("NODE_ID")
	if nodeIDStr == "" {
		return nil, fmt.Errorf("NODE_ID is not set")
	}
	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("NODE_ID is not a valid integer: %w", err)
	}
	return &StaticRegistry{nodeID: nodeID}, nil
}

func (r *StaticRegistry) Acquire(_ context.Context) (int64, error) { return r.nodeID, nil }
func (r *StaticRegistry) Release(_ context.Context) error          { return nil }
func (r *StaticRegistry) Renew(_ context.Context) error            { return nil }
func (r *StaticRegistry) NodeID() int64                            { return r.nodeID }
