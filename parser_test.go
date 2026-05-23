package csid

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewParser_InvalidFormat(t *testing.T) {
	p, err := NewParser(WithFormat(Format{1, 1, 1}))
	if !errors.Is(err, ConfigErrInvalidBitFormat) {
		t.Errorf("NewParser() = %v, want ConfigErrInvalidBitFormat", err)
	}
	if p != nil {
		t.Error("NewParser() returned non-nil parser on error")
	}
}

func TestNewParser_Valid(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewParser() returned nil parser")
	}
}

func TestParser_Parse_KnownID(t *testing.T) {
	// Use default format: shift_ts=22, shift_node=10
	epoch := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	wantTS := int64(500) // 500ms since epoch
	wantNode := int64(3)
	wantSeq := int64(7)

	// Pack: ts << 22 | node << 10 | seq
	packed := (wantTS << 22) | (wantNode << 10) | wantSeq
	id := ID(packed)

	p, err := NewParser(WithEpoch(epoch))
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	parsed := p.Parse(id)
	wantTime := epoch.Add(time.Duration(wantTS) * time.Millisecond)

	if !parsed.Timestamp.Equal(wantTime) {
		t.Errorf("Timestamp = %v, want %v", parsed.Timestamp, wantTime)
	}
	if parsed.Node != wantNode {
		t.Errorf("Node = %d, want %d", parsed.Node, wantNode)
	}
	if parsed.Sequence != wantSeq {
		t.Errorf("Sequence = %d, want %d", parsed.Sequence, wantSeq)
	}
}

func TestParser_Parse_RoundTrip(t *testing.T) {
	r := &mockRegistry{nodeID: 42}
	n, err := New(context.Background(), r)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	before := time.Now()
	id, err := n.Generate()
	after := time.Now()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	parsed := p.Parse(id)

	if parsed.Node != 42 {
		t.Errorf("Node = %d, want 42", parsed.Node)
	}
	// Allow 1s slack for timestamp comparison
	if parsed.Timestamp.Before(before.Add(-time.Second)) || parsed.Timestamp.After(after.Add(time.Second)) {
		t.Errorf("Timestamp = %v out of expected range [%v, %v]", parsed.Timestamp, before, after)
	}
	if parsed.Sequence < 0 {
		t.Errorf("Sequence = %d, want >= 0", parsed.Sequence)
	}
}
