package grpc

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/common/demopb"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
	"google.golang.org/grpc"
)

func Init(exitSig chan os.Signal) {
	ipPort := fmt.Sprintf("%s:%d", config.Config.GrpcConfig.Host, config.Config.GrpcConfig.Port)
	listener, err := net.Listen("tcp", ipPort)
	if err != nil {
		log.Logger.Fatalf("Failed to listen: %v", err)
		exitSig <- os.Interrupt
		return
	}
	// Set up gRPC options for timeout and connection pooling
	opts := []grpc.ServerOption{
		grpc.ConnectionTimeout(time.Duration(config.Config.GrpcConfig.ConnectTimeout) * time.Second), // Set a connection timeout
		grpc.MaxConcurrentStreams(uint32(config.Config.GrpcConfig.MaxPoolSize)),                      // Set maximum concurrent streams
		grpc.MaxRecvMsgSize(1024 * 1024), // Set maximum receive message size (1MB here)
		grpc.MaxSendMsgSize(1024 * 1024), // Set maximum send message size (1MB here)
	}
	grpcServer := grpc.NewServer(opts...)
	demopb.RegisterDemoServiceServer(grpcServer, &DemoService{})

	log.Logger.Infof("Server is running on %s", ipPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Logger.Fatal("Failed to serve: %v", err)
		exitSig <- os.Interrupt
	}
}
