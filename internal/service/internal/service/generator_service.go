package service

import (
	"fmt"

	"github.com/from-cero/csid"
)

// Generator ...
type Generator struct {
	n *csid.Node
}

// NewGenerator ...
func NewGenerator(n *csid.Node) *Generator {
	return &Generator{n: n}
}

// NextID ...
func (g *Generator) NextID() (csid.ID, error) {
	id, err := g.n.Generate()
	if err != nil {
		return 0, fmt.Errorf("failed to generate ID: %w", err)
	}
	return id, nil
}
