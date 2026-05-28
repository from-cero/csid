package csid

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/from-cero/csid/registry"
)

// Node is a distributed ID generator bound to a single node ID acquired from a registry.Registry.
// All methods are safe for concurrent use.
type Node struct {
	mu      sync.Mutex        // protects shared states when Node.Generate of same Node runs on multiple goroutines
	closed  bool              // indicates whether the Node is closed
	cfg     Config            // configuration for this Node
	comF    compiledFormat    // precomputed values for bit manipulation based on cfg.Format
	reg     registry.Registry // registry for acquiring and releasing nodeID ID
	epochMs int64             // epoch in milliseconds since Unix epoch
	nodeID  int64             // the identity acquired from registry
	lastMs  int64             // the timestamp in milliseconds of the last generated ID
	seq     int64             // the sequence number for IDs generated within the same millisecond
}

// New creates a new Node, acquiring a node ID from the provided registry.Registry.
// Restart safety: seq resets to 0 on each New, but a collision requires a full restart within 1ms of
// the previous run. Go runtime init alone takes >1ms, making this impossible in practice.
func New(ctx context.Context, reg registry.Registry, opt ...Option) (*Node, error) {
	cfg := applyOptions(opt)
	if err := cfg.Format.validate(); err != nil {
		return nil, err
	}
	comF := cfg.Format.compileFormat()
	epochMs := cfg.Epoch.UnixMilli()

	if reg == nil {
		return nil, ErrNilRegistry
	}
	nodeID, err := reg.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	if nodeID < 0 || nodeID > comF.maxNode {
		return nil, ErrInvalidNodeID
	}

	return &Node{
		closed:  false,
		cfg:     cfg,
		comF:    comF,
		reg:     reg,
		epochMs: epochMs,
		nodeID:  nodeID,
		lastMs:  0,
		seq:     0,
	}, nil
}

// Close shuts down the node and releases its node ID back to the registry.Registry.
// After Close, any call to Generate returns ErrNodeClosed.
func (n *Node) Close(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closed = true
	return n.reg.Release(ctx)
}

// Generate generates a new ID [timestamp | nodeID | sequence]
//
// Clock backward behavior:
//   - If the current time is less than the last generated time,
//     it means that the clock has moved backwards.
//   - If the clock has moved backwards by more than the threshold,
//     an error is returned. (infrastructure issue, clock sync issue, etc.)
//   - If the clock has moved backwards by less than the threshold,
//     the generator will wait until the clock catches up to the last generated time before generating a new ID.
//
// Sequence exhaustion behavior:
//   - If the sequence number exceeds the maximum for the current millisecond,
//     the generator will wait until the next millisecond before generating a new ID.
//   - If YieldOnExhaustion is enabled, the generator yields (runtime.Gosched) instead of sleeping,
//     allowing it to approach the theoretical maximum throughput at the cost of CPU.
//
// NOTES:
//   - In practice, clock backward and sequence exhaustion should be extremely rare if the system is properly provisioned and monitored.
//   - If it hits seq exhaustion often enough that the lock matters, it needs more nodes.
//   - That why it isn't designed to have a smarter lock or lock-free strategy and face the complexity of handling edge cases.
//   - The current design is simpler and good enough for the expected use cases.
func (n *Node) Generate() (ID, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// if node is closed, any on-flight generate calls should stop
	if n.closed {
		return 0, ErrNodeClosed
	}

	now := n.nowMs() // milliseconds since epoch
	if now < 0 {
		return 0, ErrClockBeforeEpoch
	}
	if now > n.comF.maxTimestamp {
		return 0, ErrTimestampOverflow
	}

	if now < n.lastMs { // clock backward issue
		if n.lastMs-now > n.cfg.MaxClockDrift.Milliseconds() {
			return 0, ErrClockBackward
		}
		time.Sleep(time.Duration(n.lastMs-now) * time.Millisecond)
		now = n.nowMs()
	}

	if now < 0 {
		return 0, ErrClockBeforeEpoch
	}
	// check whether now still be behind n.lastMs
	if now < n.lastMs {
		return 0, ErrClockSyncFailed
	}
	if now == n.lastMs {
		n.seq = (n.seq + 1) & n.comF.maxSeq // increase sequence and wrap around if exceeds max
		if n.seq == 0 {                     // sequence exhausted (wrapped around) for this ms
			for now <= n.lastMs {
				if n.cfg.YieldOnExhaustion {
					runtime.Gosched()
				} else {
					time.Sleep(time.Millisecond)
				}
				now = n.nowMs()
			}
		}
	} else {
		n.seq = 0
	}
	if now > n.comF.maxTimestamp {
		return 0, ErrTimestampOverflow
	}
	n.lastMs = now

	var idI64 int64
	idI64 |= (now & n.comF.maxTimestamp) << n.comF.shiftTimestamp
	idI64 |= n.nodeID << n.comF.shiftNode
	idI64 |= n.seq
	return ID(idI64), nil
}

func (n *Node) nowMs() int64 {
	return time.Now().UnixMilli() - n.epochMs
}
