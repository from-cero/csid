package main

import (
	"fmt"
	"time"

	ceroid "github.com/from-cero/cero-id"
)

const duration = 5 * time.Second

func main() {
	node, err := ceroid.NewNode()
	if err != nil {
		panic(err)
	}

	var count int64
	deadline := time.Now().Add(duration)

	for time.Now().Before(deadline) {
		if _, err := node.Generate(); err != nil {
			panic(err)
		}
		count++
	}

	fmt.Printf("duration  : %s\n", duration)
	fmt.Printf("generated : %d IDs\n", count)
	fmt.Printf("throughput: %.0f IDs/sec\n", float64(count)/duration.Seconds())
}
