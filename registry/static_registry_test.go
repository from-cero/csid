package registry

import (
	"context"
	"testing"
)

func TestNewStaticRegistry_NoEnvVar(t *testing.T) {
	t.Setenv("NODE_ID", "")
	_, err := NewStaticRegistry()
	if err == nil {
		t.Error("NewStaticRegistry() expected error when NODE_ID is unset, got nil")
	}
}

func TestNewStaticRegistry_InvalidValue(t *testing.T) {
	t.Setenv("NODE_ID", "abc")
	_, err := NewStaticRegistry()
	if err == nil {
		t.Error("NewStaticRegistry() expected error for non-integer NODE_ID, got nil")
	}
}

func TestNewStaticRegistry_Valid(t *testing.T) {
	t.Setenv("NODE_ID", "7")
	r, err := NewStaticRegistry()
	if err != nil {
		t.Fatalf("NewStaticRegistry() error = %v", err)
	}
	if r == nil {
		t.Fatal("NewStaticRegistry() returned nil")
	}
	if r.nodeID != 7 {
		t.Errorf("nodeID = %d, want 7", r.nodeID)
	}
}

func TestStaticRegistry_Acquire(t *testing.T) {
	t.Setenv("NODE_ID", "99")
	r, err := NewStaticRegistry()
	if err != nil {
		t.Fatalf("NewStaticRegistry() error = %v", err)
	}
	got, err := r.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if got != 99 {
		t.Errorf("Acquire() = %d, want 99", got)
	}
}

func TestStaticRegistry_Release(t *testing.T) {
	t.Setenv("NODE_ID", "1")
	r, err := NewStaticRegistry()
	if err != nil {
		t.Fatalf("NewStaticRegistry() error = %v", err)
	}
	if err := r.Release(context.Background()); err != nil {
		t.Errorf("Release() error = %v, want nil", err)
	}
}
