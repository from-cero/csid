package server

import (
	"google.golang.org/grpc"
)

func newGRPCServer(_ *Services) *grpc.Server {
	// TODO
	return grpc.NewServer()
}
