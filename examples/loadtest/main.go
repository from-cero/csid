package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	ceroid "github.com/from-cero/cero-id"
	"github.com/from-cero/cero-id/registry"
)

func main() {
	nodes := flag.Int("nodes", 1, "number of nodes to run in parallel")
	goroutinesPerNode := flag.Int("goroutines", 10, "goroutines per node")
	duration := flag.Duration("duration", 10*time.Second, "test duration")
	verify := flag.Bool("verify", false, "collect all IDs and check for duplicates after the run")
	scenario := flag.String("scenario", "", "preset scenario: exhaust | multinode")
	flag.Parse()

	// apply scenario presets before validation
	switch *scenario {
	case "exhaust":
		// goroutines >> maxSeq (1024) to guarantee sequence saturation every ms
		*goroutinesPerNode = 2048
		if *nodes == 0 {
			*nodes = 1
		}
		*verify = true
		fmt.Println("scenario: exhaust — saturating sequence every millisecond")
	case "multinode":
		*nodes = 8
		*goroutinesPerNode = 128
		*verify = true
		fmt.Println("scenario: multinode — 8 nodes × 128 goroutines, cross-node dedup")
	case "":
		// manual flags
	default:
		fmt.Fprintf(os.Stderr, "unknown scenario %q (valid: exhaust, multinode)\n", *scenario)
		os.Exit(1)
	}

	if *nodes < 1 || *goroutinesPerNode < 1 {
		fmt.Fprintln(os.Stderr, "nodes and goroutines must be >= 1")
		os.Exit(1)
	}

	fmt.Printf("load test  nodes=%d  goroutines/node=%d  duration=%s  verify=%v\n\n",
		*nodes, *goroutinesPerNode, *duration, *verify)

	ctx := context.Background()
	deadline := time.Now().Add(*duration)

	type nodeResult struct {
		count    int64
		errCount int64
		lats     []int64 // nanoseconds
		ids      []ceroid.ID
	}

	results := make([]nodeResult, *nodes)

	var outerWg sync.WaitGroup
	for i := range *nodes {
		outerWg.Add(1)
		go func(nodeIdx int) {
			defer outerWg.Done()

			r := &fixedRegistry{nodeID: int64(nodeIdx)}
			node, err := ceroid.New(ctx, r)
			if err != nil {
				fmt.Fprintf(os.Stderr, "node %d: NewNode failed: %v\n", nodeIdx, err)
				os.Exit(1)
			}
			defer node.Close(ctx) //nolint:errcheck

			type workerResult struct {
				ids      []ceroid.ID
				lats     []int64
				errCount int64
			}
			workers := make([]workerResult, *goroutinesPerNode)

			var innerWg sync.WaitGroup
			for w := range *goroutinesPerNode {
				innerWg.Add(1)
				go func(wIdx int) {
					defer innerWg.Done()
					var wr workerResult
					for time.Now().Before(deadline) {
						t0 := time.Now()
						id, genErr := node.Generate()
						ns := time.Since(t0).Nanoseconds()
						if genErr != nil {
							wr.errCount++
						} else {
							wr.lats = append(wr.lats, ns)
							if *verify {
								wr.ids = append(wr.ids, id)
							}
						}
					}
					workers[wIdx] = wr
				}(w)
			}

			innerWg.Wait()

			var res nodeResult
			for _, wr := range workers {
				res.errCount += wr.errCount
				res.count += int64(len(wr.lats))
				res.lats = append(res.lats, wr.lats...)
				res.ids = append(res.ids, wr.ids...)
			}
			results[nodeIdx] = res
		}(i)
	}

	outerWg.Wait()

	// aggregate
	var totalCount, totalErr int64
	var allLats []int64
	var allIDs []ceroid.ID

	for _, r := range results {
		totalCount += r.count
		totalErr += r.errCount
		allLats = append(allLats, r.lats...)
		allIDs = append(allIDs, r.ids...)
	}

	elapsed := duration.Seconds()
	throughput := float64(totalCount) / elapsed

	fmt.Printf("results\n")
	fmt.Printf("  total IDs   : %d\n", totalCount)
	fmt.Printf("  errors      : %d\n", totalErr)
	fmt.Printf("  duration    : %s\n", *duration)
	fmt.Printf("  throughput  : %.0f IDs/s  (%.2f IDs/ms)\n", throughput, throughput/1000)

	if len(allLats) > 0 {
		sort.Slice(allLats, func(i, j int) bool { return allLats[i] < allLats[j] })
		n := len(allLats)
		fmt.Printf("  latency avg : %s\n", time.Duration(meanInt64(allLats)))
		fmt.Printf("  latency p50 : %s\n", time.Duration(allLats[n*50/100]))
		fmt.Printf("  latency p95 : %s\n", time.Duration(allLats[n*95/100]))
		fmt.Printf("  latency p99 : %s\n", time.Duration(allLats[n*99/100]))
		fmt.Printf("  latency max : %s\n", time.Duration(allLats[n-1]))
	}

	if *nodes > 1 {
		fmt.Printf("\nper-node breakdown\n")
		for i, r := range results {
			tp := float64(r.count) / elapsed
			fmt.Printf("  node %4d : %10d IDs  %7.0f IDs/s  errors: %d\n",
				i, r.count, tp, r.errCount)
		}
	}

	if *verify {
		fmt.Printf("\nduplicate check — %d IDs\n", len(allIDs))
		dups := findDuplicates(allIDs)
		if len(dups) == 0 {
			fmt.Println("  PASS  no duplicates found")
		} else {
			fmt.Printf("  FAIL  %d duplicate IDs:\n", len(dups))
			for _, id := range dups {
				fmt.Printf("    %s\n", id)
			}
			os.Exit(2)
		}
	}
}

// findDuplicates returns any ID that appears more than once.
func findDuplicates(ids []ceroid.ID) []ceroid.ID {
	seen := make(map[ceroid.ID]struct{}, len(ids))
	var dups []ceroid.ID
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			dups = append(dups, id)
		} else {
			seen[id] = struct{}{}
		}
	}
	return dups
}

func meanInt64(s []int64) int64 {
	if len(s) == 0 {
		return 0
	}
	var sum float64
	for _, v := range s {
		sum += float64(v)
	}
	return int64(math.Round(sum / float64(len(s))))
}

// fixedRegistry implements registry.Registry with a pre-assigned node ID.
type fixedRegistry struct {
	nodeID int64
}

func (r *fixedRegistry) Acquire(_ context.Context) (int64, error) { return r.nodeID, nil }
func (r *fixedRegistry) Release(_ context.Context) error          { return nil }

var _ registry.Registry = (*fixedRegistry)(nil)
