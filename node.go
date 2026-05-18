package ceroid

import (
	"context"
	"sync"
	"time"

	"github.com/from-cero/cero-id/registry"
)

type Node struct {
	mu     sync.Mutex
	closed bool
	r      registry.Registry
	node   int64
	lastMs int64
	seq    int64
	cfg    Config
	c      compiled
}

func NewNode(ctx context.Context, r registry.Registry, opt ...Option) (*Node, error) {
	cfg := applyOptions(opt)
	if err := cfg.Format.validate(); err != nil {
		return nil, err
	}

	c := cfg.Format.compileFormat()

	if r == nil {
		return nil, ErrNilRegistry
	}
	nodeID, err := r.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	if nodeID < 0 || nodeID > c.maxNode {
		return nil, ErrInvalidNodeID
	}

	return &Node{
		r:    r,
		node: nodeID,
		cfg:  cfg,
		c:    c,
	}, nil
}

func (n *Node) Close(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closed = true
	return n.r.Release(ctx)
}

// Generate generates a new ID (0 | timestamp | sequence)
//
// Clock backward behavior:
//   - If the current time is less than the last generated time, it means that the clock has moved backwards.
//   - If the clock has moved backwards by more than the threshold, an error is returned. (infrastructure issue, clock sync issue, etc.)
//   - If the clock has moved backwards by less than the threshold, the generator will wait until the clock catches up to the last generated time before generating a new ID.
//
// Sequence exhaustion behavior:
//   - If the sequence number exceeds the maximum for the current millisecond, the generator will wait until the next millisecond before generating a new ID.
func (n *Node) Generate() (ID, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return 0, ErrClosed
	}

	now := n.nowMs() // milliseconds since epoch

	if now < n.lastMs { // clock backward issue
		if n.lastMs-now > n.cfg.MaxClockDrift.Milliseconds() {
			return 0, ErrClockBackward
		}
		time.Sleep(time.Duration(n.lastMs-now) * time.Millisecond)
		now = n.lastMs
	}

	if now == n.lastMs {
		n.seq = (n.seq + 1) & n.c.maxSeq
		if n.seq == 0 { // sequence exhausted for this ms
			for now <= n.lastMs {
				time.Sleep(time.Millisecond)
				now = n.nowMs()
			}
		}
	} else {
		n.seq = 0
	}
	if now < n.lastMs {
		return 0, ErrClockBackward
	}

	n.lastMs = now

	var idI64 int64
	idI64 |= (now & n.c.maxTimestamp) << n.c.shiftTimestamp
	idI64 |= n.node << n.c.shiftNode
	idI64 |= n.seq
	return ID(idI64), nil
}

func (n *Node) Parse(id ID) ParsedID {
	return parseWith(id, n.cfg.Epoch, n.c)
}

func (n *Node) nowMs() int64 {
	return time.Now().UnixMilli() - n.cfg.Epoch.UnixMilli()
}
