package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
	"github.com/google/uuid"

	ceroid "github.com/from-cero/cero-id"
	"github.com/from-cero/cero-id/registry"
)

const (
	nodes    = 8
	perNode  = 100_000
	totalIDs = nodes * perNode
)

// fixedRegistry is a Registry that returns a hard-coded node ID.
// It simulates one pod/process that has been assigned a stable node ID at deploy time.
type fixedRegistry struct{ id int64 }

func (r *fixedRegistry) Acquire(_ context.Context) (int64, error) { return r.id, nil }
func (r *fixedRegistry) Release(_ context.Context) error          { return nil }

var _ registry.Registry = (*fixedRegistry)(nil)

func runCeroIDMultinode() (time.Duration, *hdrhistogram.Histogram, bool) {
	ids := make([]ceroid.ID, totalIDs)
	latencies := make([][]int64, nodes)
	for i := range nodes {
		latencies[i] = make([]int64, perNode)
	}

	ctx := context.Background()

	// Each node simulates an independent pod with its own generator — no shared mutex.
	nodeGenerators := make([]*ceroid.Node, nodes)
	for i := range nodes {
		n, err := ceroid.New(ctx, &fixedRegistry{id: int64(i)})
		if err != nil {
			log.Fatalf("failed to create cero-id node %d: %v", i, err)
		}
		nodeGenerators[i] = n
	}

	var wg sync.WaitGroup
	start := time.Now()

	for i := range nodes {
		wg.Add(1)
		go func(nodeIdx int) {
			defer wg.Done()
			offset := nodeIdx * perNode
			lat := latencies[nodeIdx]
			gen := nodeGenerators[nodeIdx]
			for j := range perNode {
				t0 := time.Now()
				id, err := gen.Generate()
				lat[j] = time.Since(t0).Nanoseconds()
				if err != nil {
					log.Fatalf("cero-id node %d failed: %v", nodeIdx, err)
				}
				ids[offset+j] = id
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	hist := hdrhistogram.New(1, 10_000_000_000, 3)
	for _, wl := range latencies {
		for _, ns := range wl {
			if ns < 1 {
				ns = 1
			}
			_ = hist.RecordValue(ns)
		}
	}

	seen := make(map[ceroid.ID]struct{}, totalIDs)
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return elapsed, hist, true
		}
		seen[id] = struct{}{}
	}
	return elapsed, hist, false
}

func runUUIDv7Multinode() (time.Duration, *hdrhistogram.Histogram, bool) {
	ids := make([]uuid.UUID, totalIDs)
	latencies := make([][]int64, nodes)
	for i := range nodes {
		latencies[i] = make([]int64, perNode)
	}

	var wg sync.WaitGroup
	start := time.Now()

	for i := range nodes {
		wg.Add(1)
		go func(nodeIdx int) {
			defer wg.Done()
			offset := nodeIdx * perNode
			lat := latencies[nodeIdx]
			for j := range perNode {
				t0 := time.Now()
				id, err := uuid.NewV7()
				lat[j] = time.Since(t0).Nanoseconds()
				if err != nil {
					log.Fatalf("uuidv7 node %d failed: %v", nodeIdx, err)
				}
				ids[offset+j] = id
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	hist := hdrhistogram.New(1, 10_000_000_000, 3)
	for _, wl := range latencies {
		for _, ns := range wl {
			if ns < 1 {
				ns = 1
			}
			_ = hist.RecordValue(ns)
		}
	}

	seen := make(map[uuid.UUID]struct{}, totalIDs)
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return elapsed, hist, true
		}
		seen[id] = struct{}{}
	}
	return elapsed, hist, false
}

func printResult(elapsed time.Duration, hist *hdrhistogram.Histogram, dups bool) {
	dupStr := "none"
	if dups {
		dupStr = "YES (collision detected!)"
	}
	fmt.Printf("  generated  : %d IDs\n", totalIDs)
	fmt.Printf("  duration   : %s\n", elapsed)
	fmt.Printf("  throughput : %.0f IDs/s  |  %.0f IDs/ms\n",
		float64(totalIDs)/elapsed.Seconds(),
		float64(totalIDs)/float64(elapsed.Milliseconds()))
	fmt.Printf("  latency p50: %d ns\n", hist.ValueAtQuantile(50))
	fmt.Printf("  latency p99: %d ns\n", hist.ValueAtQuantile(99))
	fmt.Printf("  duplicates : %s\n", dupStr)
}

func main() {
	fmt.Printf("benchmark: %d nodes × %d IDs = %d total\n", nodes, perNode, totalIDs)
	fmt.Printf("model: each node is an independent generator (simulates separate pods)\n\n")

	fmt.Println("--- cero-id (multinode) ---")
	ceroElapsed, ceroHist, ceroDups := runCeroIDMultinode()
	printResult(ceroElapsed, ceroHist, ceroDups)

	fmt.Println()

	fmt.Println("--- UUIDv7 (multinode) ---")
	uuidElapsed, uuidHist, uuidDups := runUUIDv7Multinode()
	printResult(uuidElapsed, uuidHist, uuidDups)

	fmt.Println()
	fmt.Println("--- comparison ---")

	throughputCero := float64(totalIDs) / ceroElapsed.Seconds()
	throughputUUID := float64(totalIDs) / uuidElapsed.Seconds()

	if ceroElapsed < uuidElapsed {
		fmt.Printf("  throughput : cero-id is %.2fx faster (%.0f vs %.0f IDs/s)\n",
			throughputCero/throughputUUID, throughputCero, throughputUUID)
		fmt.Printf("  latency p50: cero-id is %.2fx lower (%d vs %d ns)\n",
			float64(uuidHist.ValueAtQuantile(50))/float64(ceroHist.ValueAtQuantile(50)),
			ceroHist.ValueAtQuantile(50), uuidHist.ValueAtQuantile(50))
		fmt.Printf("  latency p99: cero-id is %.2fx lower (%d vs %d ns)\n",
			float64(uuidHist.ValueAtQuantile(99))/float64(ceroHist.ValueAtQuantile(99)),
			ceroHist.ValueAtQuantile(99), uuidHist.ValueAtQuantile(99))
	} else {
		fmt.Printf("  throughput : UUIDv7 is %.2fx faster (%.0f vs %.0f IDs/s)\n",
			throughputUUID/throughputCero, throughputUUID, throughputCero)
		fmt.Printf("  latency p50: UUIDv7 is %.2fx lower (%d vs %d ns)\n",
			float64(ceroHist.ValueAtQuantile(50))/float64(uuidHist.ValueAtQuantile(50)),
			uuidHist.ValueAtQuantile(50), ceroHist.ValueAtQuantile(50))
		fmt.Printf("  latency p99: UUIDv7 is %.2fx lower (%d vs %d ns)\n",
			float64(ceroHist.ValueAtQuantile(99))/float64(uuidHist.ValueAtQuantile(99)),
			uuidHist.ValueAtQuantile(99), ceroHist.ValueAtQuantile(99))
	}
}
