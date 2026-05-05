package registry

import "context"

// Registry atomically assigns a unique worker ID to this node within a
// given (datacenterID, isProd) namespace.
type Registry interface {
	// Claim acquires a worker ID in [0, 127]. The returned release function
	// must be called when the generator shuts down.
	Claim(ctx context.Context, datacenterID uint8, isProd bool) (workerID uint8, release func() error, err error)
}
