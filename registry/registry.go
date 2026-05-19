package registry

import "context"

// Registry manages node ID allocation. Implementations must be safe for concurrent use.
type Registry interface {
	Acquire(ctx context.Context) (int64, error) // Acquire reserves a unique node ID for the caller and returns it.
	Release(ctx context.Context) error          // Release frees the node ID previously acquired by this instance.
}
