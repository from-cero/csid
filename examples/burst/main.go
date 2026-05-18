package main

import (
	"fmt"
	"sync"
	"time"

	ceroid "github.com/from-cero/cero-id"
)

const (
	goroutines = 8
	perWorker  = 100_000
)

func main() {
	node, err := ceroid.NewNode()
	if err != nil {
		panic(err)
	}

	total := goroutines * perWorker
	ids := make([]ceroid.ID, total)

	var wg sync.WaitGroup
	start := time.Now()

	for i := range goroutines {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			offset := workerIdx * perWorker
			for j := range perWorker {
				id, err := node.Generate()
				if err != nil {
					panic(err)
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

	fmt.Printf("generated : %d IDs\n", total)
	fmt.Printf("duration  : %s\n", elapsed)
	fmt.Printf("throughput: %.0f IDs/sec\n", float64(total)/elapsed.Seconds())
	fmt.Printf("duplicates: none\n")
}
