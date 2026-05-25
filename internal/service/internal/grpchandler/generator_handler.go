package grpchandler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	generatorv1 "github.com/from-cero/csid/service/gen/generator/v1"
)

type iGeneratorS interface {
	NextID(ctx context.Context) (string, error)
}

type Generator struct {
	generatorv1.UnimplementedGeneratorServiceServer
	genS iGeneratorS
}

func NewGenerator(genS iGeneratorS) *Generator {
	return &Generator{genS: genS}
}

func (g *Generator) NextID(ctx context.Context, _ *generatorv1.NextIDRequest) (*generatorv1.NextIDResponse, error) {
	id, err := g.genS.NextID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate id: %v", err)
	}
	return &generatorv1.NextIDResponse{Id: id}, nil
}
