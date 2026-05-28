package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/from-cero/csid"
	"github.com/from-cero/csid/registry/static"
)

func main() {
	nodeID := flag.Int64("node", 0, "static node ID to use")
	flag.Parse()

	err := os.Setenv("NODE_ID", fmt.Sprintf("%d", *nodeID))
	if err != nil {
		log.Fatalf("failed to set NODE_ID environment variable: %v", err)
	}

	ctx := context.Background()
	r, err := static.NewRegistry("NODE_ID")
	if err != nil {
		log.Fatalf("failed to create registry: %v", err)
	}
	node, err := csid.New(ctx, r)
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

	parser, err := csid.NewParser()
	if err != nil {
		fmt.Printf("failed to create parser: %v", err)
		return
	}
	fmt.Println(parser.Parse(id))
}
