package service

import (
	"context"
)

type Bar struct{}

func NewBar() *Bar {
	return &Bar{}
}

func (s *Bar) Example(_ context.Context) error {
	panic("implement me")
}
