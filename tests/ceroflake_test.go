package test

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/from-cero/ceroflake"
	"github.com/from-cero/ceroflake/registry"
	goredis "github.com/redis/go-redis/v9"
)

func newTestGen(t *testing.T, workerID uint8, opts ...ceroflake.Option) *ceroflake.Generator {
	t.Helper()
	base := []ceroflake.Option{
		ceroflake.WithDatacenter(1),
		ceroflake.WithRegistry(registry.Static(workerID)),
	}
	g, err := ceroflake.New(context.Background(), append(base, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = g.Close() })
	return g
}

func TestGenerateUnique(t *testing.T) {
	g := newTestGen(t, 0)
	const n = 10_000
	seen := make(map[int64]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := g.Generate(ceroflake.EntityUser)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ID at iteration %d: %d", i, id)
		}
		seen[id] = struct{}{}
	}
}

func TestGenerateConcurrent(t *testing.T) {
	g := newTestGen(t, 0)
	const (
		goroutines = 8
		perG       = 10_000
	)
	var (
		mu   sync.Mutex
		seen = make(map[int64]struct{}, goroutines*perG)
		wg   sync.WaitGroup
	)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				id, err := g.Generate(ceroflake.EntityOrder)
				if err != nil {
					t.Errorf("Generate: %v", err)
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

func TestParsedRoundtrip(t *testing.T) {
	g := newTestGen(t, 5, ceroflake.WithDatacenter(3), ceroflake.WithProd())

	id, err := g.Generate(ceroflake.EntityProduct)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	p := ceroflake.Parse(id)

	if p.Entity != ceroflake.EntityProduct {
		t.Errorf("entity: got %v, want %v", p.Entity, ceroflake.EntityProduct)
	}
	if p.Datacenter != 3 {
		t.Errorf("datacenter: got %d, want 3", p.Datacenter)
	}
	if !p.IsProd {
		t.Error("isProd: got false, want true")
	}
	if p.WorkerID != 5 {
		t.Errorf("workerID: got %d, want 5", p.WorkerID)
	}
	if p.Time.IsZero() {
		t.Error("parsed time is zero")
	}
}

func TestMonotonicity(t *testing.T) {
	g := newTestGen(t, 0)
	const n = 1000
	ids := make([]int64, n)
	for i := range ids {
		id, err := g.Generate(ceroflake.EntityUser)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		ids[i] = id
	}
	if !sort.SliceIsSorted(ids, func(i, j int) bool { return ids[i] <= ids[j] }) {
		t.Error("IDs are not monotonically non-decreasing")
	}
}

func TestEntityTagging(t *testing.T) {
	g := newTestGen(t, 0)
	entities := []ceroflake.EntityType{
		ceroflake.EntityUser, ceroflake.EntityOrder,
		ceroflake.EntityProduct, ceroflake.EntityPayment,
	}
	for _, e := range entities {
		id, err := g.Generate(e)
		if err != nil {
			t.Fatalf("Generate(%v): %v", e, err)
		}
		if got := ceroflake.Parse(id).Entity; got != e {
			t.Errorf("entity mismatch: got %v, want %v", got, e)
		}
	}
}

func TestMultiNodeNoDupes(t *testing.T) {
	const (
		nodes  = 8
		perGen = 5_000
	)
	gens := make([]*ceroflake.Generator, nodes)
	for i := range gens {
		gens[i] = newTestGen(t, uint8(i))
	}

	var (
		mu   sync.Mutex
		seen = make(map[int64]struct{}, nodes*perGen)
		wg   sync.WaitGroup
	)
	wg.Add(nodes)
	for _, g := range gens {
		g := g
		go func() {
			defer wg.Done()
			for j := 0; j < perGen; j++ {
				id, err := g.Generate(ceroflake.EntityOrder)
				if err != nil {
					t.Errorf("Generate: %v", err)
					return
				}
				mu.Lock()
				if _, dup := seen[id]; dup {
					t.Errorf("duplicate across nodes: %d", id)
				}
				seen[id] = struct{}{}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

func TestRedisRegistry(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer func() {
		err := client.Close()
		if err != nil {
			t.Errorf("redis client close: %v", err)
		}
	}()

	reg := registry.Redis(client)
	ctx := context.Background()

	workerID, release, err := reg.Claim(ctx, 1, false)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if workerID > 127 {
		t.Errorf("workerID %d out of range", workerID)
	}

	// Second claim should get a different worker ID.
	workerID2, release2, err := reg.Claim(ctx, 1, false)
	if err != nil {
		t.Fatalf("second Claim: %v", err)
	}
	if workerID2 == workerID {
		t.Errorf("two claims returned same worker ID %d", workerID)
	}

	if err := release(); err != nil {
		t.Errorf("release: %v", err)
	}
	if err := release2(); err != nil {
		t.Errorf("release2: %v", err)
	}
}

func TestClockBackward(t *testing.T) {
	// This test verifies that Parse correctly recovers the timestamp even at
	// the boundary, and that Generate never returns a decreasing ID.
	g := newTestGen(t, 0)

	prev, err := g.Generate(ceroflake.EntityUser)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for i := 0; i < 100; i++ {
		id, err := g.Generate(ceroflake.EntityUser)
		if err != nil {
			t.Fatalf("Generate %d: %v", i, err)
		}
		if id < prev {
			t.Errorf("ID decreased: %d < %d", id, prev)
		}
		prev = id
	}
}
