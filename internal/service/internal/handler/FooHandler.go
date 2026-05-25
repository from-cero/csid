package handler

import (
	"context"
)

// iBarS defines the business logic required by the Foo handler.
type iBarS interface {
	// * declare methods as the service grows

	Example(ctx context.Context) error
}

type Foo struct {
	barS iBarS
}

func NewFoo(barS iBarS) *Foo {
	return &Foo{barS: barS}
}
