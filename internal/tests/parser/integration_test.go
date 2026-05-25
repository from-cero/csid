package parser_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/from-cero/csid"
)

// stubRegistry is a minimal registry.Registry implementation for integration tests.
type stubRegistry struct {
	nodeID int64
}

func (s *stubRegistry) Acquire(_ context.Context) (int64, error) {
	return s.nodeID, nil
}

func (s *stubRegistry) Release(_ context.Context) error {
	return nil
}

// defaultEpoch mirrors the default epoch used by csid.applyOptions.
var defaultEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// ---- NewParser() error cases ----

func TestNewParser_InvalidFormat_BitsBelow63(t *testing.T) {
	t.Parallel()
	_, err := csid.NewParser(csid.WithFormat(csid.Format{
		TimestampBits: 10,
		NodeBits:      10,
		SequenceBits:  10, // sum = 30
	}))
	if !errors.Is(err, csid.ErrInvalidBitFormat) {
		t.Errorf("NewParser(bits=30) = %v, want ErrInvalidBitFormat", err)
	}
}

func TestNewParser_InvalidFormat_BitsAbove63(t *testing.T) {
	t.Parallel()
	_, err := csid.NewParser(csid.WithFormat(csid.Format{
		TimestampBits: 41,
		NodeBits:      12,
		SequenceBits:  11, // sum = 64
	}))
	if !errors.Is(err, csid.ErrInvalidBitFormat) {
		t.Errorf("NewParser(bits=64) = %v, want ErrInvalidBitFormat", err)
	}
}

func TestNewParser_InvalidFormat_ReturnsNilParser(t *testing.T) {
	t.Parallel()
	p, err := csid.NewParser(csid.WithFormat(csid.Format{TimestampBits: 1, NodeBits: 1, SequenceBits: 1}))
	if err == nil {
		t.Error("NewParser(invalid) expected error, got nil")
	}
	if p != nil {
		t.Error("NewParser(invalid) returned non-nil parser on error")
	}
}

func TestNewParser_InvalidFormat_AllZero(t *testing.T) {
	t.Parallel()
	_, err := csid.NewParser(csid.WithFormat(csid.Format{}))
	if !errors.Is(err, csid.ErrInvalidBitFormat) {
		t.Errorf("NewParser(bits=0) = %v, want ErrInvalidBitFormat", err)
	}
}

// ---- NewParser() happy paths ----

func TestNewParser_DefaultConfig(t *testing.T) {
	t.Parallel()
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewParser() returned nil")
	}
}

func TestNewParser_CustomFormat(t *testing.T) {
	t.Parallel()
	f := csid.Format{TimestampBits: 43, NodeBits: 10, SequenceBits: 10}
	p, err := csid.NewParser(csid.WithFormat(f))
	if err != nil {
		t.Fatalf("NewParser(custom format) error = %v", err)
	}
	if p == nil {
		t.Fatal("NewParser(custom format) returned nil")
	}
}

func TestNewParser_CustomEpoch(t *testing.T) {
	t.Parallel()
	epoch := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
	p, err := csid.NewParser(csid.WithEpoch(epoch))
	if err != nil {
		t.Fatalf("NewParser(custom epoch) error = %v", err)
	}
	if p == nil {
		t.Fatal("NewParser(custom epoch) returned nil")
	}
}

func TestNewParser_MultipleOptions(t *testing.T) {
	t.Parallel()
	f := csid.Format{TimestampBits: 43, NodeBits: 10, SequenceBits: 10}
	epoch := time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC)
	p, err := csid.NewParser(csid.WithFormat(f), csid.WithEpoch(epoch))
	if err != nil {
		t.Fatalf("NewParser(format+epoch) error = %v", err)
	}
	if p == nil {
		t.Fatal("NewParser(format+epoch) returned nil")
	}
}

// ---- Parse() special values ----

func TestParse_ZeroID_AllFieldsZero(t *testing.T) {
	t.Parallel()
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	parsed := p.Parse(csid.ID(0))
	if !parsed.Timestamp.Equal(defaultEpoch) {
		t.Errorf("Parse(0).Timestamp = %v, want epoch %v", parsed.Timestamp, defaultEpoch)
	}
	if parsed.Node != 0 {
		t.Errorf("Parse(0).Node = %d, want 0", parsed.Node)
	}
	if parsed.Sequence != 0 {
		t.Errorf("Parse(0).Sequence = %d, want 0", parsed.Sequence)
	}
}

// ---- Parse() known-value table tests ----

func TestParse_KnownValues_DefaultFormat(t *testing.T) {
	t.Parallel()
	// Default: shiftTimestamp=22, shiftNode=10, maxSeq=1023, maxNode=4095
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	cases := []struct {
		name string
		ts   int64
		node int64
		seq  int64
	}{
		{"zero all fields", 0, 0, 0},
		{"ts only", 500, 0, 0},
		{"node only", 0, 1, 0},
		{"seq only", 0, 0, 1},
		{"all small", 100, 3, 7},
		{"max seq", 100, 1, 1023},
		{"max node", 100, 4095, 0},
		{"max seq and node", 999, 4095, 1023},
		{"large ts", 100_000, 2048, 512},
		{"ts=1ms", 1, 0, 0},
		{"all max fields", 100_000, 4095, 1023},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			packed := (tc.ts << 22) | (tc.node << 10) | tc.seq
			parsed := p.Parse(csid.ID(packed))

			wantTime := defaultEpoch.Add(time.Duration(tc.ts) * time.Millisecond)
			if !parsed.Timestamp.Equal(wantTime) {
				t.Errorf("Timestamp = %v, want %v", parsed.Timestamp, wantTime)
			}
			if parsed.Node != tc.node {
				t.Errorf("Node = %d, want %d", parsed.Node, tc.node)
			}
			if parsed.Sequence != tc.seq {
				t.Errorf("Sequence = %d, want %d", parsed.Sequence, tc.seq)
			}
		})
	}
}

func TestParse_MaxSequenceField(t *testing.T) {
	t.Parallel()
	// Default: maxSeq = (1<<10)-1 = 1023
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	wantSeq := int64(1023)
	packed := (int64(500) << 22) | (int64(1) << 10) | wantSeq
	parsed := p.Parse(csid.ID(packed))
	if parsed.Sequence != wantSeq {
		t.Errorf("Sequence = %d, want %d", parsed.Sequence, wantSeq)
	}
}

func TestParse_MaxNodeField(t *testing.T) {
	t.Parallel()
	// Default: maxNode = (1<<12)-1 = 4095
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	wantNode := int64(4095)
	packed := (int64(500) << 22) | (wantNode << 10) | int64(0)
	parsed := p.Parse(csid.ID(packed))
	if parsed.Node != wantNode {
		t.Errorf("Node = %d, want %d", parsed.Node, wantNode)
	}
}

func TestParse_SequenceZeroWhenOnlyTimestampAndNodeSet(t *testing.T) {
	t.Parallel()
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	packed := (int64(200) << 22) | (int64(100) << 10)
	parsed := p.Parse(csid.ID(packed))
	if parsed.Sequence != 0 {
		t.Errorf("Sequence = %d, want 0", parsed.Sequence)
	}
}

func TestParse_NodeZeroWhenOnlyTimestampAndSeqSet(t *testing.T) {
	t.Parallel()
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	packed := (int64(200) << 22) | (int64(0) << 10) | int64(55)
	parsed := p.Parse(csid.ID(packed))
	if parsed.Node != 0 {
		t.Errorf("Node = %d, want 0", parsed.Node)
	}
}

// ---- Parse() round-trip tests ----

func TestParse_RoundTrip_Default(t *testing.T) {
	t.Parallel()
	const nodeID = int64(42)
	reg := &stubRegistry{nodeID: nodeID}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = n.Close(context.Background()) }()

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
	if parsed.Node != nodeID {
		t.Errorf("Node = %d, want %d", parsed.Node, nodeID)
	}
	slack := time.Second
	if parsed.Timestamp.Before(before.Add(-slack)) || parsed.Timestamp.After(after.Add(slack)) {
		t.Errorf("Timestamp = %v out of range [%v, %v]", parsed.Timestamp, before, after)
	}
	if parsed.Sequence < 0 {
		t.Errorf("Sequence = %d, want >= 0", parsed.Sequence)
	}
}

func TestParse_RoundTrip_CustomFormat(t *testing.T) {
	t.Parallel()
	// 43-bit timestamp, 10-bit node (maxNode=1023), 10-bit sequence
	f := csid.Format{TimestampBits: 43, NodeBits: 10, SequenceBits: 10}
	const nodeID = int64(200)
	reg := &stubRegistry{nodeID: nodeID}
	n, err := csid.New(context.Background(), reg, csid.WithFormat(f))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = n.Close(context.Background()) }()

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
		t.Errorf("Node = %d, want %d", parsed.Node, nodeID)
	}
	slack := time.Second
	if parsed.Timestamp.Before(before.Add(-slack)) || parsed.Timestamp.After(after.Add(slack)) {
		t.Errorf("Timestamp = %v out of range", parsed.Timestamp)
	}
}

func TestParse_RoundTrip_CustomEpoch(t *testing.T) {
	t.Parallel()
	customEpoch := time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)
	const nodeID = int64(10)
	reg := &stubRegistry{nodeID: nodeID}
	n, err := csid.New(context.Background(), reg, csid.WithEpoch(customEpoch))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = n.Close(context.Background()) }()

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
	if parsed.Node != nodeID {
		t.Errorf("Node = %d, want %d", parsed.Node, nodeID)
	}
	slack := time.Second
	if parsed.Timestamp.Before(before.Add(-slack)) || parsed.Timestamp.After(after.Add(slack)) {
		t.Errorf("Timestamp = %v out of range", parsed.Timestamp)
	}
}

func TestParse_RoundTrip_CustomFormatAndEpoch(t *testing.T) {
	t.Parallel()
	f := csid.Format{TimestampBits: 40, NodeBits: 13, SequenceBits: 10}
	epoch := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	const nodeID = int64(8191) // maxNode for 13 bits = (1<<13)-1 = 8191
	reg := &stubRegistry{nodeID: nodeID}
	n, err := csid.New(context.Background(), reg, csid.WithFormat(f), csid.WithEpoch(epoch))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = n.Close(context.Background()) }()

	p, err := csid.NewParser(csid.WithFormat(f), csid.WithEpoch(epoch))
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
		t.Errorf("Node = %d, want %d", parsed.Node, nodeID)
	}
	slack := time.Second
	if parsed.Timestamp.Before(before.Add(-slack)) || parsed.Timestamp.After(after.Add(slack)) {
		t.Errorf("Timestamp = %v out of range", parsed.Timestamp)
	}
}

// ---- Parse() precision and accuracy ----

func TestParse_TimestampPrecisionIsMillisecond(t *testing.T) {
	t.Parallel()
	reg := &stubRegistry{nodeID: 1}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = n.Close(context.Background()) }()

	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	id, err := n.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	parsed := p.Parse(id)
	// Timestamp has millisecond precision: nanosecond offset within its second
	// must be a multiple of 1,000,000 (no sub-ms component).
	if parsed.Timestamp.Nanosecond()%1_000_000 != 0 {
		t.Errorf("Timestamp.Nanosecond() = %d not a multiple of 1ms", parsed.Timestamp.Nanosecond())
	}
}

func TestParse_TimestampReflectsCustomEpoch(t *testing.T) {
	t.Parallel()
	// Pack a known offset into the ID and verify the epoch shifts the recovered time
	epoch1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	epoch2 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	wantTS := int64(1000)
	packed := wantTS << 22 // node=0, seq=0

	p1, err := csid.NewParser(csid.WithEpoch(epoch1))
	if err != nil {
		t.Fatalf("NewParser(epoch1) error = %v", err)
	}
	p2, err := csid.NewParser(csid.WithEpoch(epoch2))
	if err != nil {
		t.Fatalf("NewParser(epoch2) error = %v", err)
	}

	parsed1 := p1.Parse(csid.ID(packed))
	parsed2 := p2.Parse(csid.ID(packed))

	expectedDiff := epoch1.Sub(epoch2)
	actualDiff := parsed1.Timestamp.Sub(parsed2.Timestamp)
	if actualDiff != expectedDiff {
		t.Errorf("epoch shift: got diff %v, want %v", actualDiff, expectedDiff)
	}
}

// ---- Parse() batch and concurrent tests ----

func TestParse_BatchRoundTrip(t *testing.T) {
	t.Parallel()
	const nodeID = int64(99)
	reg := &stubRegistry{nodeID: nodeID}
	n, err := csid.New(context.Background(), reg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = n.Close(context.Background()) }()

	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	const count = 300
	for i := 0; i < count; i++ {
		id, err := n.Generate()
		if err != nil {
			t.Fatalf("Generate() at i=%d error = %v", i, err)
		}
		parsed := p.Parse(id)
		if parsed.Node != nodeID {
			t.Errorf("i=%d: parsed.Node = %d, want %d", i, parsed.Node, nodeID)
		}
		if parsed.Sequence < 0 {
			t.Errorf("i=%d: parsed.Sequence = %d, want >= 0", i, parsed.Sequence)
		}
	}
}

func TestParse_Concurrent_SafeForParallelUse(t *testing.T) {
	t.Parallel()
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	// Pre-build known IDs to avoid depending on generator concurrency
	const count = 100
	type expected struct {
		id       csid.ID
		wantTS   time.Time
		wantNode int64
		wantSeq  int64
	}
	items := make([]expected, count)
	for i := range items {
		ts := int64(i + 1)
		node := int64(i % 20)
		seq := int64(i % 50)
		packed := (ts << 22) | (node << 10) | seq
		items[i] = expected{
			id:       csid.ID(packed),
			wantTS:   defaultEpoch.Add(time.Duration(ts) * time.Millisecond),
			wantNode: node,
			wantSeq:  seq,
		}
	}

	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		i := i
		go func() {
			defer wg.Done()
			parsed := p.Parse(items[i].id)
			if !parsed.Timestamp.Equal(items[i].wantTS) {
				t.Errorf("i=%d: Timestamp = %v, want %v", i, parsed.Timestamp, items[i].wantTS)
			}
			if parsed.Node != items[i].wantNode {
				t.Errorf("i=%d: Node = %d, want %d", i, parsed.Node, items[i].wantNode)
			}
			if parsed.Sequence != items[i].wantSeq {
				t.Errorf("i=%d: Sequence = %d, want %d", i, parsed.Sequence, items[i].wantSeq)
			}
		}()
	}
	wg.Wait()
}

// ---- Parse() custom format known-value tests ----

func TestParse_CustomFormat_KnownValues(t *testing.T) {
	t.Parallel()
	// Format{43, 10, 10}: shiftTimestamp=20, shiftNode=10, maxNode=1023, maxSeq=1023
	f := csid.Format{TimestampBits: 43, NodeBits: 10, SequenceBits: 10}
	epoch := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p, err := csid.NewParser(csid.WithFormat(f), csid.WithEpoch(epoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	cases := []struct {
		name string
		ts   int64
		node int64
		seq  int64
	}{
		{"small values", 750, 512, 99},
		{"max node", 1000, 1023, 0},
		{"max seq", 1000, 0, 1023},
		{"all max", 1_000_000, 1023, 1023},
		{"zero all", 0, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// shiftTimestamp=20, shiftNode=10 for this format
			packed := (tc.ts << 20) | (tc.node << 10) | tc.seq
			parsed := p.Parse(csid.ID(packed))
			wantTime := epoch.Add(time.Duration(tc.ts) * time.Millisecond)
			if !parsed.Timestamp.Equal(wantTime) {
				t.Errorf("Timestamp = %v, want %v", parsed.Timestamp, wantTime)
			}
			if parsed.Node != tc.node {
				t.Errorf("Node = %d, want %d", parsed.Node, tc.node)
			}
			if parsed.Sequence != tc.seq {
				t.Errorf("Sequence = %d, want %d", parsed.Sequence, tc.seq)
			}
		})
	}
}

func TestParse_SequenceFieldFullRange(t *testing.T) {
	t.Parallel()
	// Verify every sequence value from 0 to maxSeq decodes correctly
	// Default: maxSeq = 1023 (10 bits)
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	const maxSeq = int64(1023)
	for seq := int64(0); seq <= maxSeq; seq++ {
		packed := (int64(100) << 22) | (int64(1) << 10) | seq
		parsed := p.Parse(csid.ID(packed))
		if parsed.Sequence != seq {
			t.Errorf("Sequence = %d, want %d", parsed.Sequence, seq)
		}
	}
}

func TestParse_NodeFieldAllValidValues(t *testing.T) {
	t.Parallel()
	// Spot-check several node values across the full range for default format (0..4095)
	p, err := csid.NewParser(csid.WithEpoch(defaultEpoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	spots := []int64{0, 1, 255, 1024, 2047, 4094, 4095}
	for _, node := range spots {
		packed := (int64(500) << 22) | (node << 10) | int64(0)
		parsed := p.Parse(csid.ID(packed))
		if parsed.Node != node {
			t.Errorf("Node = %d, want %d", parsed.Node, node)
		}
	}
}

func TestParse_IDFromMultipleNodes_CorrectNodeDecoded(t *testing.T) {
	t.Parallel()
	// Generate IDs from different nodes and verify each ID decodes to the right node
	p, err := csid.NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	nodes := []int64{0, 1, 7, 42, 100, 4095}
	for _, nodeID := range nodes {
		reg := &stubRegistry{nodeID: nodeID}
		n, err := csid.New(context.Background(), reg)
		if err != nil {
			t.Fatalf("New(nodeID=%d) error = %v", nodeID, err)
		}
		id, err := n.Generate()
		_ = n.Close(context.Background())
		if err != nil {
			t.Fatalf("Generate(nodeID=%d) error = %v", nodeID, err)
		}
		parsed := p.Parse(id)
		if parsed.Node != nodeID {
			t.Errorf("nodeID=%d: parsed.Node = %d", nodeID, parsed.Node)
		}
	}
}
