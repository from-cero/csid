package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"

	"github.com/from-cero/csid"
)

const (
	goroutines = 8
	perWorker  = 100_000
)

func run(busySpin bool) (time.Duration, *hdrhistogram.Histogram, bool) {
	total := goroutines * perWorker
	ids := make([]csid.ID, total)
	latencies := make([][]int64, goroutines)
	for i := range goroutines {
		latencies[i] = make([]int64, perWorker)
	}

	ctx := context.Background()
	node, err := csid.New(ctx, &fixedRegistry{}, csid.WithBusySpin(busySpin))
	if err != nil {
		log.Fatalf("failed to create node: %v", err)
	}
	defer node.Close(ctx)

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
					fmt.Printf("worker %d failed: %v\n", workerIdx, genErr)
					return
				}
				ids[offset+j] = id
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	seen := make(map[csid.ID]struct{}, total)
	dups := false
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			fmt.Printf("DUPLICATE: %d\n", id)
			dups = true
		}
		seen[id] = struct{}{}
	}

	hist := hdrhistogram.New(1, 10_000_000_000, 3)
	for _, workerLat := range latencies {
		for _, ns := range workerLat {
			if ns < 1 {
				ns = 1
			}
			_ = hist.RecordValue(ns)
		}
	}

	return elapsed, hist, dups
}

func printResult(label string, elapsed time.Duration, hist *hdrhistogram.Histogram, dups bool) {
	total := goroutines * perWorker
	fmt.Printf("--- %s ---\n", label)
	fmt.Printf("generated : %d IDs\n", total)
	fmt.Printf("duration  : %s\n", elapsed)
	fmt.Printf(
		"throughput: %.0f IDs/s, %.0f IDs/ms\n",
		float64(total)/elapsed.Seconds(),
		float64(total)/float64(elapsed.Milliseconds()),
	)
	if dups {
		fmt.Printf("duplicates: YES\n")
	} else {
		fmt.Printf("duplicates: none\n")
	}
	fmt.Printf("latency p50: %d ns\n", hist.ValueAtQuantile(50))
	fmt.Printf("latency p99: %d ns\n", hist.ValueAtQuantile(99))
}

func main() {
	total := goroutines * perWorker
	fmt.Printf("benchmark: %d goroutines x %d IDs = %d total\n\n", goroutines, perWorker, total)

	sleepElapsed, sleepHist, sleepDups := run(false)
	printResult("sleep (default)", sleepElapsed, sleepHist, sleepDups)

	fmt.Println()

	spinElapsed, spinHist, spinDups := run(true)
	printResult("busy spin", spinElapsed, spinHist, spinDups)

	fmt.Println()
	fmt.Println("--- comparison ---")

	sleepThroughput := float64(total) / sleepElapsed.Seconds()
	spinThroughput := float64(total) / spinElapsed.Seconds()

	if spinElapsed < sleepElapsed {
		fmt.Printf(
			"  throughput : busy spin is %.2fx faster (%.0f vs %.0f IDs/s)\n",
			spinThroughput/sleepThroughput, spinThroughput, sleepThroughput,
		)
		fmt.Printf(
			"  latency p50: busy spin is %.2fx lower (%d vs %d ns)\n",
			float64(sleepHist.ValueAtQuantile(50))/float64(spinHist.ValueAtQuantile(50)),
			spinHist.ValueAtQuantile(50), sleepHist.ValueAtQuantile(50),
		)
		fmt.Printf(
			"  latency p99: busy spin is %.2fx lower (%d vs %d ns)\n",
			float64(sleepHist.ValueAtQuantile(99))/float64(spinHist.ValueAtQuantile(99)),
			spinHist.ValueAtQuantile(99), sleepHist.ValueAtQuantile(99),
		)
	} else {
		fmt.Printf(
			"  throughput : sleep is %.2fx faster (%.0f vs %.0f IDs/s)\n",
			sleepThroughput/spinThroughput, sleepThroughput, spinThroughput,
		)
	}
}

type fixedRegistry struct{}

func (r *fixedRegistry) Acquire(_ context.Context) (int64, error) { return 0, nil }
func (r *fixedRegistry) Release(_ context.Context) error          { return nil }
