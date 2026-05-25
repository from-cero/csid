package csid

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type mockRegistry struct {
	nodeID     int64
	acquireErr error
	released   bool
}

func (m *mockRegistry) Acquire(_ context.Context) (int64, error) {
	return m.nodeID, m.acquireErr
}

func (m *mockRegistry) Release(_ context.Context) error {
	m.released = true
	return nil
}

func newTestNode(t *testing.T, r *mockRegistry, opts ...Option) *Node {
	t.Helper()
	n, err := New(context.Background(), r, opts...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return n
}

func TestNew_NilRegistry(t *testing.T) {
	_, err := New(context.Background(), nil)
	if !errors.Is(err, ErrNilRegistry) {
		t.Errorf("New(nil) = %v, want ErrNilRegistry", err)
	}
}

func TestNew_InvalidFormat(t *testing.T) {
	r := &mockRegistry{nodeID: 0}
	_, err := New(context.Background(), r, WithFormat(Format{1, 1, 1}))
	if !errors.Is(err, ErrInvalidBitFormat) {
		t.Errorf("New(bad format) = %v, want ErrInvalidBitFormat", err)
	}
}

func TestNew_AcquireError(t *testing.T) {
	wantErr := errors.New("acquire failed")
	r := &mockRegistry{acquireErr: wantErr}
	_, err := New(context.Background(), r)
	if !errors.Is(err, wantErr) {
		t.Errorf("New() = %v, want %v", err, wantErr)
	}
}

func TestNew_NodeIDOutOfRange(t *testing.T) {
	// Default format: maxNode = (1<<12)-1 = 4095
	r := &mockRegistry{nodeID: 4096}
	_, err := New(context.Background(), r)
	if !errors.Is(err, ErrInvalidNodeID) {
		t.Errorf("New(nodeID=4096) = %v, want ErrInvalidNodeID", err)
	}
}

func TestNew_Success(t *testing.T) {
	r := &mockRegistry{nodeID: 1}
	n, err := New(context.Background(), r)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if n == nil {
		t.Fatal("New() returned nil node")
	}
}

func TestNode_CloseReleasesRegistry(t *testing.T) {
	r := &mockRegistry{nodeID: 1}
	n := newTestNode(t, r)
	if err := n.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !r.released {
		t.Error("Close() did not release registry")
	}
}

func TestNode_GenerateAfterClose(t *testing.T) {
	r := &mockRegistry{nodeID: 1}
	n := newTestNode(t, r)
	_ = n.Close(context.Background())
	_, err := n.Generate()
	if !errors.Is(err, ErrNodeClosed) {
		t.Errorf("Generate() after Close = %v, want ErrNodeClosed", err)
	}
}

func TestNode_Generate_Normal(t *testing.T) {
	r := &mockRegistry{nodeID: 1}
	n := newTestNode(t, r)
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if id == 0 {
		t.Error("Generate() returned zero ID")
	}
}

func TestNode_Generate_BeforeEpoch(t *testing.T) {
	r := &mockRegistry{nodeID: 1}
	// Epoch set far in the future -> nowMs() will be negative
	futureEpoch := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	n := newTestNode(t, r, WithEpoch(futureEpoch))
	_, err := n.Generate()
	if !errors.Is(err, ErrClockBeforeEpoch) {
		t.Errorf("Generate() = %v, want ErrClockBeforeEpoch", err)
	}
}

func TestNode_Generate_TimestampOverflow(t *testing.T) {
	// 3-bit timestamp -> maxTimestamp = 7ms
	// Epoch 100 years ago -> ms count will be enormous
	r := &mockRegistry{nodeID: 0}
	oldEpoch := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	f := Format{TimestampBits: 3, NodeBits: 10, SequenceBits: 50}
	n := newTestNode(t, r, WithFormat(f), WithEpoch(oldEpoch))
	_, err := n.Generate()
	if !errors.Is(err, ErrTimestampOverflow) {
		t.Errorf("Generate() = %v, want ErrTimestampOverflow", err)
	}
}

func TestNode_Generate_ClockBackward_BeyondTolerance(t *testing.T) {
	r := &mockRegistry{nodeID: 1}
	n := newTestNode(t, r, WithMaxClockDrift(1*time.Millisecond))

	// Simulate clock jump: set lastMs to far ahead of current time
	n.lastMs = n.nowMs() + 1000

	_, err := n.Generate()
	if !errors.Is(err, ErrClockBackward) {
		t.Errorf("Generate() = %v, want ErrClockBackward", err)
	}
}

func TestNode_Generate_ClockBackward_WithinTolerance(t *testing.T) {
	r := &mockRegistry{nodeID: 1}
	n := newTestNode(t, r, WithMaxClockDrift(50*time.Millisecond))

	// Set lastMs to 2ms ahead -- within tolerance, generator sleeps and retries
	n.lastMs = n.nowMs() + 2

	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v (within tolerance should succeed)", err)
	}
	if id == 0 {
		t.Error("Generate() returned zero ID")
	}
}

func TestNode_Generate_Concurrent(t *testing.T) {
	r := &mockRegistry{nodeID: 5}
	n := newTestNode(t, r)

	const goroutines = 10
	const perGoroutine = 100

	var mu sync.Mutex
	seen := make(map[ID]struct{}, goroutines*perGoroutine)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				id, err := n.Generate()
				if err != nil {
					t.Errorf("Generate() error = %v", err)
					return
				}
				mu.Lock()
				if _, dup := seen[id]; dup {
					t.Errorf("duplicate ID: %d", id)
				}
				seen[id] = struct{}{}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

func TestNode_Generate_IDComponents(t *testing.T) {
	r := &mockRegistry{nodeID: 7}
	n := newTestNode(t, r)

	before := time.Now()
	id, err := n.Generate()
	after := time.Now()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	p := NewParserFromNode(n)
	parsed := p.Parse(id)

	if parsed.Node != 7 {
		t.Errorf("parsed.Node = %d, want 7", parsed.Node)
	}
	if parsed.Timestamp.Before(before.Add(-time.Second)) || parsed.Timestamp.After(after.Add(time.Second)) {
		t.Errorf(
			"parsed.Timestamp = %v, expected between %v and %v", parsed.Timestamp, before, after,
		)
	}
	if parsed.Sequence < 0 {
		t.Errorf("parsed.Sequence = %d, want >= 0", parsed.Sequence)
	}
}

// NewParserFromNode creates a Parser using the same config as an existing Node (test helper).
func NewParserFromNode(n *Node) *Parser {
	return &Parser{cfg: n.cfg, comF: n.comF}
}
