package server

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	generatorv1 "github.com/from-cero/csid/service/gen/generator/v1"
	"github.com/from-cero/csid/service/internal/config"
)

func newGRPCServer(_ *config.GRPCConfig, h *Handlers) *grpc.Server {
	s := grpc.NewServer()
	generatorv1.RegisterGeneratorServiceServer(s, h.genGRPCH)
	reflection.Register(s)
	return s
}
