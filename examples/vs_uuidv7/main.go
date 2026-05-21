package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/from-cero/cero-id/registry"
	"github.com/google/uuid"

	"github.com/from-cero/csid"
)

const (
	goroutines = 8
	perWorker  = 100_000
)

func runCeroID() (time.Duration, *hdrhistogram.Histogram, bool) {
	err := os.Setenv("NODE_ID", "0")
	if err != nil {
		log.Fatalf("failed to set NODE_ID: %v", err)
	}

	ctx := context.Background()
	r, err := registry.NewStaticRegistry()
	if err != nil {
		log.Fatalf("failed to create registry: %v", err)
	}
	node, err := csid.New(ctx, r)
	if err != nil {
		log.Fatalf("failed to create node: %v", err)
	}

	total := goroutines * perWorker
	ids := make([]csid.ID, total)
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
					log.Fatalf("cero-id worker %d failed: %v", workerIdx, genErr)
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

	seen := make(map[csid.ID]struct{}, total)
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return elapsed, hist, true
		}
		seen[id] = struct{}{}
	}
	return elapsed, hist, false
}

func runUUIDv7() (time.Duration, *hdrhistogram.Histogram, bool) {
	total := goroutines * perWorker
	ids := make([]uuid.UUID, total)
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
				id, genErr := uuid.NewV7()
				lat[j] = time.Since(t0).Nanoseconds()
				if genErr != nil {
					log.Fatalf("uuidv7 worker %d failed: %v", workerIdx, genErr)
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

	seen := make(map[uuid.UUID]struct{}, total)
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return elapsed, hist, true
		}
		seen[id] = struct{}{}
	}
	return elapsed, hist, false
}

func printResult(name string, elapsed time.Duration, hist *hdrhistogram.Histogram, dups bool) {
	total := goroutines * perWorker
	dupStr := "none"
	if dups {
		dupStr = "YES (collision detected!)"
	}
	fmt.Printf("  generated  : %d IDs\n", total)
	fmt.Printf("  duration   : %s\n", elapsed)
	fmt.Printf(
		"  throughput : %.0f IDs/s  |  %.0f IDs/ms\n",
		float64(total)/elapsed.Seconds(),
		float64(total)/float64(elapsed.Milliseconds()),
	)
	fmt.Printf("  latency p50: %d ns\n", hist.ValueAtQuantile(50))
	fmt.Printf("  latency p99: %d ns\n", hist.ValueAtQuantile(99))
	fmt.Printf("  duplicates : %s\n", dupStr)
	_ = name
}

func main() {
	total := goroutines * perWorker
	fmt.Printf("benchmark: %d goroutines × %d IDs = %d total\n\n", goroutines, perWorker, total)

	fmt.Println("--- cero-id ---")
	ceroElapsed, ceroHist, ceroDups := runCeroID()
	printResult("cero-id", ceroElapsed, ceroHist, ceroDups)

	fmt.Println()

	fmt.Println("--- UUIDv7 ---")
	uuidElapsed, uuidHist, uuidDups := runUUIDv7()
	printResult("UUIDv7", uuidElapsed, uuidHist, uuidDups)

	fmt.Println()
	fmt.Println("--- comparison ---")

	throughputCero := float64(total) / ceroElapsed.Seconds()
	throughputUUID := float64(total) / uuidElapsed.Seconds()

	if ceroElapsed < uuidElapsed {
		fmt.Printf(
			"  throughput : cero-id is %.2fx faster (%.0f vs %.0f IDs/s)\n",
			throughputCero/throughputUUID, throughputCero, throughputUUID,
		)
		fmt.Printf(
			"  latency p50: cero-id is %.2fx lower (%d vs %d ns)\n",
			float64(uuidHist.ValueAtQuantile(50))/float64(ceroHist.ValueAtQuantile(50)),
			ceroHist.ValueAtQuantile(50), uuidHist.ValueAtQuantile(50),
		)
		fmt.Printf(
			"  latency p99: cero-id is %.2fx lower (%d vs %d ns)\n",
			float64(uuidHist.ValueAtQuantile(99))/float64(ceroHist.ValueAtQuantile(99)),
			ceroHist.ValueAtQuantile(99), uuidHist.ValueAtQuantile(99),
		)
	} else {
		fmt.Printf(
			"  throughput : UUIDv7 is %.2fx faster (%.0f vs %.0f IDs/s)\n",
			throughputUUID/throughputCero, throughputUUID, throughputCero,
		)
		fmt.Printf(
			"  latency p50: UUIDv7 is %.2fx lower (%d vs %d ns)\n",
			float64(ceroHist.ValueAtQuantile(50))/float64(uuidHist.ValueAtQuantile(50)),
			uuidHist.ValueAtQuantile(50), ceroHist.ValueAtQuantile(50),
		)
		fmt.Printf(
			"  latency p99: UUIDv7 is %.2fx lower (%d vs %d ns)\n",
			float64(ceroHist.ValueAtQuantile(99))/float64(uuidHist.ValueAtQuantile(99)),
			uuidHist.ValueAtQuantile(99), ceroHist.ValueAtQuantile(99),
		)
	}
}
