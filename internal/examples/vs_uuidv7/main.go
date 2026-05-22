package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
	"github.com/google/uuid"

	"github.com/from-cero/csid"
)

func runCSID(nodeCount, targetPerNode int, yieldOnExhaustion bool) (time.Duration, *hdrhistogram.Histogram, bool) {
	total := nodeCount * targetPerNode
	ids := make([]csid.ID, total)
	latencies := make([][]int64, nodeCount)
	for i := range nodeCount {
		latencies[i] = make([]int64, targetPerNode)
	}

	ctx := context.Background()
	nodes := make([]*csid.Node, nodeCount)
	for i := range nodeCount {
		n, err := csid.New(ctx, &fixedRegistry{id: int64(i)}, csid.WithYieldOnExhaustion(yieldOnExhaustion))
		if err != nil {
			log.Fatalf("failed to create csid node %d: %v", i, err)
		}
		nodes[i] = n
	}

	var wg sync.WaitGroup
	start := time.Now()

	for n := range nodeCount {
		wg.Add(1)
		go func(nodeIdx int, node *csid.Node) {
			defer wg.Done()
			offset := nodeIdx * targetPerNode
			lat := latencies[nodeIdx]
			for j := range targetPerNode {
				t0 := time.Now()
				id, err := node.Generate()
				lat[j] = time.Since(t0).Nanoseconds()
				if err != nil {
					log.Fatalf("csid worker %d failed: %v", nodeIdx, err)
				}
				ids[offset+j] = id
			}
		}(n, nodes[n])
	}

	wg.Wait()
	elapsed := time.Since(start)

	for _, n := range nodes {
		err := n.Close(ctx)
		if err != nil {
			log.Printf("failed to close csid node: %v", err)
		}
	}

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

func runUUIDv7(count, targetPerNode int) (time.Duration, *hdrhistogram.Histogram, bool) {
	total := count * targetPerNode
	ids := make([]uuid.UUID, total)
	latencies := make([][]int64, count)
	for i := range count {
		latencies[i] = make([]int64, targetPerNode)
	}

	var wg sync.WaitGroup
	start := time.Now()

	for i := range count {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			offset := workerIdx * targetPerNode
			lat := latencies[workerIdx]
			for j := range targetPerNode {
				t0 := time.Now()
				id, err := uuid.NewV7()
				lat[j] = time.Since(t0).Nanoseconds()
				if err != nil {
					log.Fatalf("uuidv7 worker %d failed: %v", workerIdx, err)
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

func printResult(elapsed time.Duration, hist *hdrhistogram.Histogram, total int, dups bool) {
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
}

func main() {
	csidMultiNode := flag.Bool("csid-multi-node", false, "use one csid node per goroutine instead of sharing one node")
	uuidMultiNode := flag.Bool("uuidv7-multi-node", false, "run UUIDv7 with multiple goroutines instead of one")
	nodes := flag.Int("nodes", 8, "number of nodes/goroutines (used when the respective multi-node flag is set)")
	targetPerNode := flag.Int("target-per-node", 100_000, "IDs to generate per node/goroutine")
	yieldOnExhaustion := flag.Bool("yield-on-exhaustion", false, "yield instead of sleep when sequence is exhausted")
	flag.Parse()

	csidCount := 1
	if *csidMultiNode {
		csidCount = *nodes
	}
	uuidCount := 1
	if *uuidMultiNode {
		uuidCount = *nodes
	}

	csidTotal := csidCount * *targetPerNode
	uuidTotal := uuidCount * *targetPerNode

	var csidLabel, uuidLabel string
	if *csidMultiNode {
		csidLabel = fmt.Sprintf("csid (%d nodes, 1 goroutine each)", csidCount)
	} else {
		csidLabel = "csid (1 node)"
	}
	if *uuidMultiNode {
		uuidLabel = fmt.Sprintf("UUIDv7 (%d goroutines, shared global generator)", uuidCount)
	} else {
		uuidLabel = "UUIDv7 (1 goroutine, shared global generator)"
	}

	fmt.Printf(
		"csid-multi-node: %v, uuidv7-multi-node: %v, nodes: %d, target-per-node: %d, yield-on-exhaustion: %v\n\n",
		*csidMultiNode,
		*uuidMultiNode,
		*nodes,
		*targetPerNode,
		*yieldOnExhaustion,
	)

	fmt.Printf("--- %s ---\n", csidLabel)
	ceroElapsed, ceroHist, ceroDups := runCSID(csidCount, *targetPerNode, *yieldOnExhaustion)
	printResult(ceroElapsed, ceroHist, csidTotal, ceroDups)

	fmt.Println()

	fmt.Printf("--- %s ---\n", uuidLabel)
	uuidElapsed, uuidHist, uuidDups := runUUIDv7(uuidCount, *targetPerNode)
	printResult(uuidElapsed, uuidHist, uuidTotal, uuidDups)

	fmt.Println()
	fmt.Println("--- comparison ---")

	throughputCero := float64(csidTotal) / ceroElapsed.Seconds()
	throughputUUID := float64(uuidTotal) / uuidElapsed.Seconds()

	if ceroElapsed < uuidElapsed {
		fmt.Printf(
			"  throughput : csid is %.2fx faster (%.0f vs %.0f IDs/s)\n",
			throughputCero/throughputUUID, throughputCero, throughputUUID,
		)
		fmt.Printf(
			"  latency p50: csid is %.2fx lower (%d vs %d ns)\n",
			float64(uuidHist.ValueAtQuantile(50))/float64(ceroHist.ValueAtQuantile(50)),
			ceroHist.ValueAtQuantile(50), uuidHist.ValueAtQuantile(50),
		)
		fmt.Printf(
			"  latency p99: csid is %.2fx lower (%d vs %d ns)\n",
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

type fixedRegistry struct{ id int64 }

func (r *fixedRegistry) Acquire(_ context.Context) (int64, error) { return r.id, nil }
func (r *fixedRegistry) Release(_ context.Context) error          { return nil }
