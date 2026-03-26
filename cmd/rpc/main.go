package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Neo4717/NeoCoin/config"
	"github.com/Neo4717/NeoCoin/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, logger.TextEncoding, "", "", false)
	logger.Info("NeoCoin gRPC gateway starting...")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.RPCPort))
	if err != nil {
		logger.Error("Failed to listen: %v", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	reflection.Register(s)

	logger.Info("gRPC gateway listening on :%d", cfg.RPCPort)

	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Error("gRPC error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down gRPC gateway...")
	s.GracefulStop()
	logger.Info("gRPC gateway stopped")
}
