package registry

import "context"

type Registry interface {
	Acquire(ctx context.Context) (int64, error)
	Release(ctx context.Context) error
	Renew(ctx context.Context) error
	NodeID() int64
}
