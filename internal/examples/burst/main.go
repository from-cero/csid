package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"

	"github.com/from-cero/csid"
)

func main() {
	nodes := flag.Int("nodes", 1, "number of nodes to simulate")
	goroutinesPerNode := flag.Int("goroutines-per-node", 1, "goroutines per node")
	targetPerNode := flag.Int("target-per-node", 100_000, "target IDs to generate per node")
	yieldOnExhaustion := flag.Bool("yield-on-exhaustion", false, "yield instead of sleep when sequence is exhausted")
	flag.Parse()

	if *targetPerNode%*goroutinesPerNode != 0 {
		log.Fatalf(
			"target-per-node (%d) must be divisible by goroutines-per-node (%d)",
			*targetPerNode,
			*goroutinesPerNode,
		)
	}
	perWorker := *targetPerNode / *goroutinesPerNode

	fmt.Printf(
		"nodes: %d, goroutines-per-node: %d, target-per-node: %d, yield-on-exhaustion: %v\n",
		*nodes, *goroutinesPerNode, *targetPerNode, *yieldOnExhaustion,
	)
	fmt.Printf("target: %d\n\n", *nodes**targetPerNode)

	ctx := context.Background()

	totalWorkers := *nodes * *goroutinesPerNode
	total := totalWorkers * perWorker
	ids := make([]csid.ID, total)
	latencies := make([][]int64, totalWorkers)
	for i := range totalWorkers {
		latencies[i] = make([]int64, perWorker)
	}

	nodeList := make([]*csid.Node, *nodes)
	for n := range *nodes {
		node, err := csid.New(ctx, &fixedRegistry{id: int64(n)}, csid.WithYieldOnExhaustion(*yieldOnExhaustion))
		if err != nil {
			log.Fatalf("failed to create node %d: %v", n, err)
		}
		nodeList[n] = node
	}

	var wg sync.WaitGroup
	start := time.Now()

	for n := range *nodes {
		for g := range *goroutinesPerNode {
			workerIdx := n**goroutinesPerNode + g
			wg.Add(1)
			go func(workerIdx int, node *csid.Node) {
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
			}(workerIdx, nodeList[n])
		}
	}

	wg.Wait()
	elapsed := time.Since(start)

	for _, node := range nodeList {
		err := node.Close(ctx)
		if err != nil {
			log.Fatalf("failed to close node %v: %v", node, err)
		}
	}

	seen := make(map[csid.ID]struct{}, total)
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			fmt.Printf("DUPLICATE: %d\n", id)
			return
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

	fmt.Printf("generated : %d IDs\n", total)
	fmt.Printf("duration  : %s\n", elapsed)
	fmt.Printf(
		"throughput: %.0f IDs/s, %.0f IDs/ms\n",
		float64(total)/elapsed.Seconds(),
		float64(total)/float64(elapsed.Milliseconds()),
	)
	fmt.Printf("duplicates: none\n")
	fmt.Printf("latency p50: %d ns\n", hist.ValueAtQuantile(50))
	fmt.Printf("latency p99: %d ns\n", hist.ValueAtQuantile(99))
}

type fixedRegistry struct{ id int64 }

func (r *fixedRegistry) Acquire(_ context.Context) (int64, error) { return r.id, nil }
func (r *fixedRegistry) Release(_ context.Context) error          { return nil }
