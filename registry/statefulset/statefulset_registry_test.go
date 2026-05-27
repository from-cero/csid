package statefulset_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/from-cero/csid/registry/statefulset"
)

func newTestRegistry(t *testing.T, maxNode int64, opts ...statefulset.Option) *statefulset.Registry {
	t.Helper()
	reg, err := statefulset.NewRegistry(maxNode, opts...)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return reg
}

func TestNewRegistry_NegativeMaxNode(t *testing.T) {
	_, err := statefulset.NewRegistry(-1)
	if !errors.Is(err, statefulset.ErrInvalidMaxNodeID) {
		t.Errorf("error = %v, want ErrInvalidMaxNodeID", err)
	}
}

func TestAcquire_ParsesOrdinal(t *testing.T) {
	cases := []struct {
		podName string
		want    int64
	}{
		{"myapp-0", 0},
		{"myapp-3", 3},
		{"my-app-name-7", 7},
		{"x-100", 100},
	}
	for _, tc := range cases {
		t.Run(tc.podName, func(t *testing.T) {
			reg := newTestRegistry(t, 4095, statefulset.WithPodName(tc.podName))
			id, err := reg.Acquire(context.Background())
			if err != nil {
				t.Fatalf("Acquire() error = %v", err)
			}
			if id != tc.want {
				t.Errorf("Acquire() = %d, want %d", id, tc.want)
			}
			_ = reg.Release(context.Background())
		})
	}
}

func TestAcquire_InvalidHostname(t *testing.T) {
	invalidNames := []string{"", "nohyphen", "trailing-", "myapp-abc", "myapp-1.0"}
	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			reg := newTestRegistry(t, 4095, statefulset.WithPodName(name))
			_, err := reg.Acquire(context.Background())
			if !errors.Is(err, statefulset.ErrInvalidHostname) {
				t.Errorf("Acquire(%q) error = %v, want ErrInvalidHostname", name, err)
			}
		})
	}
}

func TestAcquire_OrdinalOutOfRange(t *testing.T) {
	reg := newTestRegistry(t, 3, statefulset.WithPodName("myapp-5"))
	_, err := reg.Acquire(context.Background())
	if !errors.Is(err, statefulset.ErrOrdinalOutOfRange) {
		t.Errorf("error = %v, want ErrOrdinalOutOfRange", err)
	}
}

func TestAcquire_OrdinalAtMaxNode(t *testing.T) {
	reg := newTestRegistry(t, 5, statefulset.WithPodName("myapp-5"))
	id, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if id != 5 {
		t.Errorf("Acquire() = %d, want 5", id)
	}
	_ = reg.Release(context.Background())
}

func TestAcquire_Idempotent(t *testing.T) {
	reg := newTestRegistry(t, 4095, statefulset.WithPodName("myapp-7"))

	id1, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	id2, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("second Acquire() error = %v", err)
	}
	if id1 != id2 {
		t.Errorf("idempotent Acquire: first=%d, second=%d; want same", id1, id2)
	}
	_ = reg.Release(context.Background())
}

func TestRelease_BeforeAcquire(t *testing.T) {
	reg := newTestRegistry(t, 4095, statefulset.WithPodName("myapp-1"))
	err := reg.Release(context.Background())
	if !errors.Is(err, statefulset.ErrNotAcquired) {
		t.Errorf("error = %v, want ErrNotAcquired", err)
	}
}

func TestRelease_AllowsReacquire(t *testing.T) {
	reg := newTestRegistry(t, 4095, statefulset.WithPodName("myapp-2"))

	if _, err := reg.Acquire(context.Background()); err != nil {
		t.Fatalf("first Acquire() error = %v", err)
	}
	if err := reg.Release(context.Background()); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	id, err := reg.Acquire(context.Background())
	if err != nil {
		t.Fatalf("second Acquire() after Release error = %v", err)
	}
	if id != 2 {
		t.Errorf("second Acquire() = %d, want 2", id)
	}
	_ = reg.Release(context.Background())
}

func TestAcquire_PodNameFuncError(t *testing.T) {
	sentinelErr := fmt.Errorf("no hostname")
	reg := newTestRegistry(t, 4095, statefulset.WithPodNameFunc(func() (string, error) {
		return "", sentinelErr
	}))
	_, err := reg.Acquire(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinelErr) {
		t.Errorf("error = %v, want to wrap sentinelErr", err)
	}
}

func TestAcquire_ConcurrentCallsSameID(t *testing.T) {
	reg := newTestRegistry(t, 4095, statefulset.WithPodName("myapp-4"))

	const n = 20
	ids := make([]int64, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ids[idx], errs[idx] = reg.Acquire(context.Background())
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Errorf("goroutine %d Acquire() error = %v", i, errs[i])
			continue
		}
		if ids[i] != 4 {
			t.Errorf("goroutine %d Acquire() = %d, want 4", i, ids[i])
		}
	}
	_ = reg.Release(context.Background())
}
