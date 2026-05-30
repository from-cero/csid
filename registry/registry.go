package registry

import "context"

// Registry manages node ID allocation. Implementations must be safe for concurrent use.
type Registry interface {
	// Acquire reserves a unique node ID for the caller and returns it.
	Acquire(ctx context.Context) (nodeID int64, err error)

	// Release frees the node ID previously acquired by this instance.
	Release(ctx context.Context) error
}
