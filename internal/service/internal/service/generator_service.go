package service

import (
	"context"

	"github.com/from-cero/csid"
)

type Generator struct {
	n *csid.Node
}

func NewGenerator(n *csid.Node) *Generator {
	return &Generator{n: n}
}

func (s *Generator) NextID(_ context.Context) (string, error) {
	id, err := s.n.Generate()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
