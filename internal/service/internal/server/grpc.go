package server

import (
	"google.golang.org/grpc"

	"github.com/from-cero/csid/service/internal/config"
)

// TODO
func newGRPCServer(_ *config.GRPCConfig, _ *Services) *grpc.Server {
	return grpc.NewServer()
}
