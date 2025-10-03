package grpc

import (
	"context"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/common/demopb"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
)

type DemoService struct {
	demopb.UnimplementedDemoServiceServer
}

func (s *DemoService) SayHello(ctx context.Context, in *demopb.HelloRequest) (*demopb.HelloResponse, error) {
	log.Logger.Infof("Received: %v", in.GetName())
	return &demopb.HelloResponse{Message: "Hello " + in.GetName()}, nil
}
