package registry

import "context"

// Registry atomically assigns a unique worker ID to this node within a
// (datacenterID, isProd) namespace.
type Registry interface {
	// Claim acquires a worker ID. The returned release function must be called
	// when the generator shuts down to free the slot.
	Claim(ctx context.Context, datacenterID uint8, isProd bool) (workerID uint8, release func() error, err error)
}
