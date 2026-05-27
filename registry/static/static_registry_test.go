package static

import (
	"context"
	"errors"
	"testing"
)

func TestNewRegistry_NotSet(t *testing.T) {
	t.Setenv("NODE_ID", "")
	_, err := NewRegistry("NODE_ID")
	if !errors.Is(err, ErrEnvNodeIDNotSet) {
		t.Errorf("NewRegistry() error = %v, want ErrEnvNodeIDNotSet", err)
	}
}

func TestNewRegistry_InvalidValue(t *testing.T) {
	t.Setenv("NODE_ID", "abc")
	_, err := NewRegistry("NODE_ID")
	if !errors.Is(err, ErrInvalidEnvNodeID) {
		t.Errorf("NewRegistry() error = %v, want ErrInvalidEnvNodeID", err)
	}
}

func TestNewRegistry_NegativeValue(t *testing.T) {
	t.Setenv("NODE_ID", "-1")
	_, err := NewRegistry("NODE_ID")
	if !errors.Is(err, ErrInvalidEnvNodeID) {
		t.Errorf("NewRegistry() error = %v, want ErrInvalidEnvNodeID", err)
	}
}

func TestNewRegistry_EmptyKeyDefaultsToNodeID(t *testing.T) {
	t.Setenv("NODE_ID", "42")
	r, err := NewRegistry("")
	if err != nil {
		t.Fatalf("NewRegistry(\"\") error = %v", err)
	}
	if r.nodeID != 42 {
		t.Errorf("nodeID = %d, want 42", r.nodeID)
	}
}

func TestNewRegistry_CustomEnvKey(t *testing.T) {
	t.Setenv("MY_NODE_ID", "7")
	r, err := NewRegistry("MY_NODE_ID")
	if err != nil {
		t.Fatalf("NewRegistry(\"MY_NODE_ID\") error = %v", err)
	}
	if r.nodeID != 7 {
		t.Errorf("nodeID = %d, want 7", r.nodeID)
	}
}

func TestNewRegistry_CustomEnvKey_NotSet(t *testing.T) {
	t.Setenv("MY_NODE_ID", "")
	_, err := NewRegistry("MY_NODE_ID")
	if !errors.Is(err, ErrEnvNodeIDNotSet) {
		t.Errorf("NewRegistry(\"MY_NODE_ID\") error = %v, want ErrEnvNodeIDNotSet", err)
	}
}

func TestNewRegistry_Valid(t *testing.T) {
	t.Setenv("NODE_ID", "7")
	r, err := NewRegistry("NODE_ID")
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if r.nodeID != 7 {
		t.Errorf("nodeID = %d, want 7", r.nodeID)
	}
}

func TestRegistry_Acquire(t *testing.T) {
	t.Setenv("NODE_ID", "99")
	r, err := NewRegistry("NODE_ID")
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	got, err := r.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if got != 99 {
		t.Errorf("Acquire() = %d, want 99", got)
	}
}

func TestRegistry_Release(t *testing.T) {
	t.Setenv("NODE_ID", "1")
	r, err := NewRegistry("NODE_ID")
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if err := r.Release(context.Background()); err != nil {
		t.Errorf("Release() error = %v, want nil", err)
	}
}
