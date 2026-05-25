package generator_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/from-cero/csid"
)

// stubRegistry is a minimal registry.Registry implementation for integration tests.
type stubRegistry struct {
	nodeID     int64
	acquireErr error
	released   bool
	releaseErr error
}

func (s *stubRegistry) Acquire(_ context.Context) (int64, error) {
	return s.nodeID, s.acquireErr
}

func (s *stubRegistry) Release(_ context.Context) error {
	s.released = true
	return s.releaseErr
}

func newNode(t *testing.T, reg *stubRegistry, opts ...csid.Option) *csid.Node {
	t.Helper()
	n, err := csid.New(context.Background(), reg, opts...)
	if err != nil {
		t.Fatalf("csid.New() unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = n.Close(context.Background()) })
	return n
}

// ---- New() error cases ----

func TestNew_NilRegistry(t *testing.T) {
	t.Parallel()
	_, err := csid.New(context.Background(), nil)
	if !errors.Is(err, csid.ErrNilRegistry) {
		t.Errorf("New(nil) = %v, want ErrNilRegistry", err)
	}
}

func TestNew_InvalidFormat_BitsBelow63(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 0}
	_, err := csid.New(context.Background(), reg, csid.WithFormat(csid.Format{
		TimestampBits: 10,
		NodeBits:      10,
		SequenceBits:  10, // sum = 30
	}))
	if !errors.Is(err, csid.ErrInvalidBitFormat) {
		t.Errorf("New(bits=30) = %v, want ErrInvalidBitFormat", err)
	}
}

func TestNew_InvalidFormat_BitsAbove63(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 0}
	_, err := csid.New(context.Background(), reg, csid.WithFormat(csid.Format{
		TimestampBits: 41,
		NodeBits:      12,
		SequenceBits:  11, // sum = 64
	}))
	if !errors.Is(err, csid.ErrInvalidBitFormat) {
		t.Errorf("New(bits=64) = %v, want ErrInvalidBitFormat", err)
	}
}

func TestNew_InvalidFormat_AllZero(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 0}
	_, err := csid.New(context.Background(), reg, csid.WithFormat(csid.Format{}))
	if !errors.Is(err, csid.ErrInvalidBitFormat) {
		t.Errorf("New(bits=0) = %v, want ErrInvalidBitFormat", err)
	}
}

func TestNew_AcquireError_Propagated(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("registry unavailable")
	reg := &stubRegistry{acquireErr: sentinel}
	_, err := csid.New(context.Background(), reg)
	if !errors.Is(err, sentinel) {
		t.Errorf("New() = %v, want %v", err, sentinel)
	}
}

func TestNew_NodeIDNegative(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: -1}
	_, err := csid.New(context.Background(), reg)
	if !errors.Is(err, csid.ErrInvalidNodeID) {
		t.Errorf("New(nodeID=-1) = %v, want ErrInvalidNodeID", err)
	}
}

func TestNew_NodeIDExceedsDefaultFormatMax(t *testing.T) {
	t.Parallel()
	// Default NodeBits=12: maxNode = (1<<12)-1 = 4095
	reg := &stubRegistry{nodeID: 4096}
	_, err := csid.New(context.Background(), reg)
	if !errors.Is(err, csid.ErrInvalidNodeID) {
		t.Errorf("New(nodeID=4096) = %v, want ErrInvalidNodeID", err)
	}
}

func TestNew_NodeIDExceedsCustomFormatMax(t *testing.T) {
	t.Parallel()
	// NodeBits=4: maxNode = (1<<4)-1 = 15
	reg := &stubRegistry{nodeID: 16}
	f := csid.Format{TimestampBits: 50, NodeBits: 4, SequenceBits: 9}
	_, err := csid.New(context.Background(), reg, csid.WithFormat(f))
	if !errors.Is(err, csid.ErrInvalidNodeID) {
		t.Errorf("New(nodeID=16, maxNode=15) = %v, want ErrInvalidNodeID", err)
	}
}

func TestNew_NodeIDAtMax_Succeeds(t *testing.T) {
	t.Parallel()
	// Default: maxNode = 4095
	reg := &stubRegistry{nodeID: 4095}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New(nodeID=4095) error = %v", err)
	}
	if n == nil {
		t.Fatal("New() returned nil")
	}
	_ = n.Close(context.Background())
}

func TestNew_NodeIDZero_Succeeds(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 0}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New(nodeID=0) error = %v", err)
	}
	if n == nil {
		t.Fatal("New() returned nil")
	}
	_ = n.Close(context.Background())
}

func TestNew_CustomFormat_Valid(t *testing.T) {
	t.Parallel()
	// 43-bit timestamp, 10-bit node, 10-bit sequence
	reg := &stubRegistry{nodeID: 5}
	f := csid.Format{TimestampBits: 43, NodeBits: 10, SequenceBits: 10}
	n, err := csid.New(context.Background(), reg, csid.WithFormat(f))
	if err != nil {
		t.Fatalf("New(custom format) error = %v", err)
	}
	if n == nil {
		t.Fatal("New(custom format) returned nil")
	}
	_ = n.Close(context.Background())
}

// ---- Close() ----

func TestClose_ReleasesRegistry(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := n.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !reg.released {
		t.Error("Close() did not call Release on registry")
	}
}

func TestClose_GenerateReturnsErrNodeClosed(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := n.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	_, err = n.Generate()
	if !errors.Is(err, csid.ErrNodeClosed) {
		t.Errorf("Generate() after Close = %v, want ErrNodeClosed", err)
	}
}

func TestClose_MultipleGeneratesAfterClose(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_ = n.Close(context.Background())
	for i := 0; i < 5; i++ {
		_, err := n.Generate()
		if !errors.Is(err, csid.ErrNodeClosed) {
			t.Errorf("Generate()[%d] after Close = %v, want ErrNodeClosed", i, err)
		}
	}
}

// ---- Generate() error cases ----

func TestGenerate_BeforeEpoch(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	futureEpoch := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	n := newNode(t, reg, csid.WithEpoch(futureEpoch))
	_, err := n.Generate()
	if !errors.Is(err, csid.ErrClockBeforeEpoch) {
		t.Errorf("Generate() with future epoch = %v, want ErrClockBeforeEpoch", err)
	}
}

func TestGenerate_TimestampOverflow(t *testing.T) {
	t.Parallel()
	// TimestampBits=3: maxTimestamp=7ms; epoch 100+ years ago guarantees overflow
	reg := &stubRegistry{nodeID: 0}
	oldEpoch := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	f := csid.Format{TimestampBits: 3, NodeBits: 10, SequenceBits: 50}
	n := newNode(t, reg, csid.WithFormat(f), csid.WithEpoch(oldEpoch))
	_, err := n.Generate()
	if !errors.Is(err, csid.ErrTimestampOverflow) {
		t.Errorf("Generate() overflow = %v, want ErrTimestampOverflow", err)
	}
}

// ErrClockBackward and ErrClockSyncFailed require manipulating internal state (lastMs)
// which is not accessible from outside the package. Those are covered by unit tests.

// ---- Generate() happy paths ----

func TestGenerate_ReturnsNonZeroID(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if id == 0 {
		t.Error("Generate() returned zero ID (timestamp should make it non-zero)")
	}
}

func TestGenerate_IDsAreUniqueSequential(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	const count = 2000
	seen := make(map[csid.ID]struct{}, count)
	for i := 0; i < count; i++ {
		id, err := n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v", i, err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ID at i=%d: %v", i, id)
		}
		seen[id] = struct{}{}
	}
}

func TestGenerate_IDsAreMonotonicallyIncreasing(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	const count = 500
	prev, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	for i := 1; i < count; i++ {
		id, err := n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v", i, err)
		}
		if id <= prev {
			t.Fatalf("ID[%d]=%v <= ID[%d]=%v: not monotonically increasing", i, id, i-1, prev)
		}
		prev = id
	}
}

func TestGenerate_IDsAreUniqueConcurrent(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 5}
	n := newNode(t, reg)
	const goroutines = 20
	const perGoroutine = 200
	var mu sync.Mutex
	seen := make(map[csid.ID]struct{}, goroutines*perGoroutine)
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
					t.Errorf("duplicate ID: %v", id)
				}
				seen[id] = struct{}{}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

func TestGenerate_ConcurrentWithClose(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 3}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, err := n.Generate()
				if err != nil {
					if errors.Is(err, csid.ErrNodeClosed) {
						return
					}
					t.Errorf("Generate() unexpected error = %v", err)
					return
				}
			}
		}()
	}
	time.Sleep(time.Millisecond)
	_ = n.Close(context.Background())
	wg.Wait()
}

func TestGenerate_NodeFieldEncoded(t *testing.T) {
	t.Parallel()
	const wantNode = int64(42)
	reg := &stubRegistry{nodeID: wantNode}
	n := newNode(t, reg)
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	parsed := p.Parse(id)
	if parsed.Node != wantNode {
		t.Errorf("parsed.Node = %d, want %d", parsed.Node, wantNode)
	}
}

func TestGenerate_NodeIDZero_FieldEncoded(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 0}
	n := newNode(t, reg)
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	parsed := p.Parse(id)
	if parsed.Node != 0 {
		t.Errorf("parsed.Node = %d, want 0", parsed.Node)
	}
}

func TestGenerate_NodeIDAtMax_FieldEncoded(t *testing.T) {
	t.Parallel()
	// Default: maxNode = 4095
	const wantNode = int64(4095)
	reg := &stubRegistry{nodeID: wantNode}
	n := newNode(t, reg)
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	parsed := p.Parse(id)
	if parsed.Node != wantNode {
		t.Errorf("parsed.Node = %d, want %d", parsed.Node, wantNode)
	}
}

func TestGenerate_TimestampWithinRange(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	before := time.Now()
	id, err := n.Generate()
	after := time.Now()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	parsed := p.Parse(id)
	slack := time.Second
	if parsed.Timestamp.Before(before.Add(-slack)) || parsed.Timestamp.After(after.Add(slack)) {
		t.Errorf("parsed.Timestamp = %v, expected between %v and %v", parsed.Timestamp, before, after)
	}
}

func TestGenerate_SequenceNonNegative(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	for i := 0; i < 200; i++ {
		id, err := n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v", i, err)
		}
		parsed := p.Parse(id)
		if parsed.Sequence < 0 {
			t.Errorf("i=%d: parsed.Sequence = %d, want >= 0", i, parsed.Sequence)
		}
	}
}

func TestGenerate_SequenceWithinMaxForFormat(t *testing.T) {
	t.Parallel()
	// Default: maxSeq = (1<<10)-1 = 1023
	const maxSeq = int64(1023)
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	for i := 0; i < 200; i++ {
		id, err := n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v", i, err)
		}
		parsed := p.Parse(id)
		if parsed.Sequence > maxSeq {
			t.Errorf("i=%d: parsed.Sequence = %d exceeds maxSeq %d", i, parsed.Sequence, maxSeq)
		}
	}
}

func TestGenerate_CustomEpoch_TimestampEncoded(t *testing.T) {
	t.Parallel()
	customEpoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg, csid.WithEpoch(customEpoch))
	p, err := csid.NewParser(csid.WithEpoch(customEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	before := time.Now()
	id, err := n.Generate()
	after := time.Now()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	parsed := p.Parse(id)
	slack := time.Second
	if parsed.Timestamp.Before(before.Add(-slack)) || parsed.Timestamp.After(after.Add(slack)) {
		t.Errorf("custom epoch: parsed.Timestamp = %v, want in [%v, %v]", parsed.Timestamp, before, after)
	}
}

func TestGenerate_CustomFormat_RoundTrip(t *testing.T) {
	t.Parallel()
	// 43-bit timestamp, 10-bit node (maxNode=1023), 10-bit sequence
	f := csid.Format{TimestampBits: 43, NodeBits: 10, SequenceBits: 10}
	const nodeID = int64(5)
	reg := &stubRegistry{nodeID: nodeID}
	n := newNode(t, reg, csid.WithFormat(f))
	p, err := csid.NewParser(csid.WithFormat(f))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	before := time.Now()
	id, err := n.Generate()
	after := time.Now()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	parsed := p.Parse(id)
	if parsed.Node != nodeID {
		t.Errorf("parsed.Node = %d, want %d", parsed.Node, nodeID)
	}
	slack := time.Second
	if parsed.Timestamp.Before(before.Add(-slack)) || parsed.Timestamp.After(after.Add(slack)) {
		t.Errorf("custom format: parsed.Timestamp = %v out of range", parsed.Timestamp)
	}
}

func TestGenerate_YieldOnExhaustion_NoError(t *testing.T) {
	t.Parallel()
	// SequenceBits=2: maxSeq=3, so only 4 IDs/ms before exhaustion
	f := csid.Format{TimestampBits: 51, NodeBits: 10, SequenceBits: 2}
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg, csid.WithFormat(f), csid.WithYieldOnExhaustion(true))
	for i := 0; i < 20; i++ {
		_, err := n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v (YieldOnExhaustion)", i, err)
		}
	}
}

func TestGenerate_SleepOnExhaustion_NoError(t *testing.T) {
	t.Parallel()
	// SequenceBits=2: maxSeq=3, so only 4 IDs/ms before exhaustion
	f := csid.Format{TimestampBits: 51, NodeBits: 10, SequenceBits: 2}
	reg := &stubRegistry{nodeID: 1}
	// YieldOnExhaustion=false (default): sleeps until next ms
	n := newNode(t, reg, csid.WithFormat(f))
	for i := 0; i < 12; i++ {
		_, err := n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v (sleep on exhaustion)", i, err)
		}
	}
}

func TestGenerate_HighVolume_AllUnique(t *testing.T) {
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	const count = 10_000
	seen := make(map[csid.ID]struct{}, count)
	for i := 0; i < count; i++ {
		id, err := n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v", i, err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ID at i=%d: %v", i, id)
		}
		seen[id] = struct{}{}
	}
}

func TestGenerate_MultipleNodes_IDsAreUnique(t *testing.T) {
	t.Parallel()
	// Two nodes with different IDs should never produce the same ID
	const count = 500
	reg1 := &stubRegistry{nodeID: 10}
	reg2 := &stubRegistry{nodeID: 20}
	n1 := newNode(t, reg1)
	n2 := newNode(t, reg2)
	seen := make(map[csid.ID]struct{}, count*2)
	var mu sync.Mutex
	var wg sync.WaitGroup
	generate := func(n *csid.Node) {
		defer wg.Done()
		for i := 0; i < count; i++ {
			id, err := n.Generate()
			if err != nil {
				t.Errorf("Generate() error = %v", err)
				return
			}
			mu.Lock()
			if _, dup := seen[id]; dup {
				t.Errorf("cross-node duplicate ID: %v", id)
			}
			seen[id] = struct{}{}
			mu.Unlock()
		}
	}
	wg.Add(2)
	go generate(n1)
	go generate(n2)
	wg.Wait()
}

func TestGenerate_MaxClockDrift_CustomValue(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg, csid.WithMaxClockDrift(50*time.Millisecond))
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if id == 0 {
		t.Error("Generate() returned zero ID")
	}
}

func TestGenerate_FullLifecycle(t *testing.T) {
	t.Parallel()
	// Create node, generate IDs, parse them, close node - all should succeed
	reg := &stubRegistry{nodeID: 7}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	ids := make([]csid.ID, 100)
	for i := range ids {
		ids[i], err = n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v", i, err)
		}
	}
	for i, id := range ids {
		parsed := p.Parse(id)
		if parsed.Node != 7 {
			t.Errorf("i=%d: parsed.Node = %d, want 7", i, parsed.Node)
		}
	}
	if err := n.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !reg.released {
		t.Error("registry was not released after Close()")
	}
	_, err = n.Generate()
	if !errors.Is(err, csid.ErrNodeClosed) {
		t.Errorf("Generate() after Close = %v, want ErrNodeClosed", err)
	}
}

// ---- ID type methods ----

func TestID_Int64(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if id.Int64() != int64(id) {
		t.Errorf("Int64() = %d, want %d", id.Int64(), int64(id))
	}
	if id.Int64() == 0 {
		t.Error("Int64() returned 0")
	}
}

func TestID_String(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n := newNode(t, reg)
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	s := id.String()
	if s == "" {
		t.Error("String() returned empty string")
	}
	if s == "0" {
		t.Error("String() returned '0' for a generated ID")
	}
	// String() must be consistent with Int64()
	if id.String() != csid.ID(id.Int64()).String() {
		t.Error("String() is not consistent with Int64()")
	}
}

func TestID_MarshalJSON_QuotedDecimalString(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 3}
	n := newNode(t, reg)
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	b, err := id.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	// Must be a quoted decimal string: "12345"
	if len(b) < 3 || b[0] != '"' || b[len(b)-1] != '"' {
		t.Errorf("MarshalJSON() = %s, want quoted decimal string", b)
	}
	// Content between quotes must match String()
	inner := string(b[1 : len(b)-1])
	if inner != id.String() {
		t.Errorf("MarshalJSON() inner = %q, want %q", inner, id.String())
	}
}

func TestID_UnmarshalJSON_RoundTrip(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 3}
	n := newNode(t, reg)
	orig, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	data, err := orig.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var got csid.ID
	if err := got.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if got != orig {
		t.Errorf("round-trip: got %v, want %v", got, orig)
	}
}

func TestID_UnmarshalJSON_ViaStdlib(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 2}
	n := newNode(t, reg)
	orig, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	type wrapper struct {
		ID csid.ID `json:"id"`
	}
	encoded, err := json.Marshal(wrapper{ID: orig})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var decoded wrapper
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.ID != orig {
		t.Errorf("json round-trip: got %v, want %v", decoded.ID, orig)
	}
}

func TestID_UnmarshalJSON_InvalidQuoting(t *testing.T) {
	t.Parallel()
	var id csid.ID
	// Unquoted number is not a valid encoded ID
	if err := id.UnmarshalJSON([]byte("12345")); err == nil {
		t.Error("UnmarshalJSON(unquoted) expected error, got nil")
	}
}

func TestID_UnmarshalJSON_NonNumericContent(t *testing.T) {
	t.Parallel()
	var id csid.ID
	if err := id.UnmarshalJSON([]byte(`"abc"`)); err == nil {
		t.Error("UnmarshalJSON(non-numeric) expected error, got nil")
	}
}

// ---- ParsedID.String ----

func TestParsedID_String_ContainsComponents(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 7}
	n := newNode(t, reg)
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	parsed := p.Parse(id)
	s := parsed.String()
	if s == "" {
		t.Error("ParsedID.String() returned empty string")
	}
	// String must contain the node value "7"
	found := false
	needle := "7"
	for i := 0; i <= len(s)-len(needle); i++ {
		if s[i:i+len(needle)] == needle {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ParsedID.String() = %q does not contain node value %q", s, needle)
	}
}

// ---- NewParser error path ----

func TestNewParser_InvalidFormat_FromGeneratorSuite(t *testing.T) {
	t.Parallel()
	// Exercises the NewParser validate-and-return-nil path.
	_, err := csid.NewParser(csid.WithFormat(csid.Format{
		TimestampBits: 20,
		NodeBits:      20,
		SequenceBits:  20, // sum = 60, not 63
	}))
	if !errors.Is(err, csid.ErrInvalidBitFormat) {
		t.Errorf("NewParser(invalid) = %v, want ErrInvalidBitFormat", err)
	}
}
