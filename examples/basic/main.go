package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	ceroid "github.com/from-cero/cero-id"
	"github.com/from-cero/cero-id/registry"
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
	defer func() {
		err := node.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close node: %v", err)
		}
	}()

	id, err := node.Generate()
	if err != nil {
		fmt.Printf("failed to generate ID: %v", err)
		return
	}
	fmt.Printf("%s -> ", id.String())

	parser, err := ceroid.NewParser()
	if err != nil {
		fmt.Printf("failed to create parser: %v", err)
		return
	}
	fmt.Println(parser.Parse(id))
}
