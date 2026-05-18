package main

import (
	"fmt"

	ceroid "github.com/from-cero/cero-id"
)

func main() {
	node, err := ceroid.NewNode()
	if err != nil {
		panic(err)
	}
	id, err := node.Generate()
	if err != nil {
		panic(err)
	}
	fmt.Println(id)

	parser, err := ceroid.NewParser()
	if err != nil {
		panic(err)
	}
	fmt.Println(parser.Parse(id))
}
