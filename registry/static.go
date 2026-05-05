package registry

import (
	"context"
	"fmt"
)

// Static returns a Registry that always claims the given fixed worker ID.
// Useful for testing or single-node deployments where IDs are assigned manually.
func Static(workerID uint8) Registry {
	return &staticRegistry{id: workerID}
}

type staticRegistry struct {
	id uint8
}

func (s *staticRegistry) Claim(_ context.Context, _ uint8, _ bool) (uint8, func() error, error) {
	if s.id > 127 {
		return 0, nil, fmt.Errorf("ceroflake/registry: static worker ID %d exceeds max 127", s.id)
	}
	return s.id, func() error { return nil }, nil
}
