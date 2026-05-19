package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"

	ceroid "github.com/from-cero/cero-id"
	"github.com/from-cero/cero-id/registry"
)

const (
	goroutines = 8
	perWorker  = 100_000
)

func main() {
	nodeID := flag.Int64("node", 0, "static node ID to use")
	flag.Parse()

	err := os.Setenv("NODE_ID", fmt.Sprintf("%d", *nodeID))
	if err != nil {
		log.Fatalf("failed to set NODE_ID environment variable: %v", err)
	}

	ctx := context.Background()
	r, err := registry.NewStaticRegistry()
	if err != nil {
		log.Fatalf("failed to create registry: %v", err)
	}
	node, err := ceroid.New(ctx, r)
	if err != nil {
		log.Fatalf("failed to create node: %v", err)
	}

	total := goroutines * perWorker
	ids := make([]ceroid.ID, total)
	// per-goroutine latency slices to avoid shared-state overhead during generation
	latencies := make([][]int64, goroutines)
	for i := range goroutines {
		latencies[i] = make([]int64, perWorker)
	}

	var wg sync.WaitGroup
	start := time.Now()

	for i := range goroutines {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			offset := workerIdx * perWorker
			lat := latencies[workerIdx]
			for j := range perWorker {
				t0 := time.Now()
				id, genErr := node.Generate()
				lat[j] = time.Since(t0).Nanoseconds()
				if genErr != nil {
					fmt.Printf("worker %d failed to generate ID: %v", workerIdx, genErr)
					return
				}
				ids[offset+j] = id
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	seen := make(map[ceroid.ID]struct{}, total)
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			fmt.Printf("DUPLICATE: %d\n", id)
			return
		}
		seen[id] = struct{}{}
	}

	// 1 ns to 10 s range, 3 significant figures
	hist := hdrhistogram.New(1, 10_000_000_000, 3)
	for _, workerLat := range latencies {
		for _, ns := range workerLat {
			if ns < 1 {
				ns = 1
			}
			_ = hist.RecordValue(ns)
		}
	}

	fmt.Printf("generated : %d IDs\n", total)
	fmt.Printf("duration  : %s\n", elapsed)
	fmt.Printf("throughput: %.0f IDs/s, %.0f IDs/ms\n",
		float64(total)/elapsed.Seconds(),
		float64(total)/float64(elapsed.Milliseconds()))
	fmt.Printf("duplicates: none\n")
	fmt.Printf("latency p50: %d ns\n", hist.ValueAtQuantile(50))
	fmt.Printf("latency p99: %d ns\n", hist.ValueAtQuantile(99))
}
