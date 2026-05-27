package static

import (
	"context"
	"testing"
)

func TestNewRegistry_NoEnvVar(t *testing.T) {
	t.Setenv("NODE_ID", "")
	_, err := NewRegistry()
	if err == nil {
		t.Error("NewRegistry() expected error when NODE_ID is unset, got nil")
	}
}

func TestNewRegistry_InvalidValue(t *testing.T) {
	t.Setenv("NODE_ID", "abc")
	_, err := NewRegistry()
	if err == nil {
		t.Error("NewRegistry() expected error for non-integer NODE_ID, got nil")
	}
}

func TestNewRegistry_Valid(t *testing.T) {
	t.Setenv("NODE_ID", "7")
	r, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if r.nodeID != 7 {
		t.Errorf("nodeID = %d, want 7", r.nodeID)
	}
}

func TestRegistry_Acquire(t *testing.T) {
	t.Setenv("NODE_ID", "99")
	r, err := NewRegistry()
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
	r, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if err := r.Release(context.Background()); err != nil {
		t.Errorf("Release() error = %v, want nil", err)
	}
}
