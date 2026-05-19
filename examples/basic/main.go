package main

import (
	"context"
	"fmt"
	"log"

	ceroid "github.com/from-cero/cero-id"
	"github.com/from-cero/cero-id/registry"
)

func main() {
	ctx := context.Background()
	r, err := registry.NewStaticRegistry()
	if err != nil {
		panic(err)
	}
	node, err := ceroid.New(ctx, r)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := node.Close(ctx)
		if err != nil {
			log.Println(err)
		}
	}()

	id, err := node.Generate()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s -> ", id.String())

	parser, err := ceroid.NewParser()
	if err != nil {
		panic(err)
	}
	fmt.Println(parser.Parse(id))
}
