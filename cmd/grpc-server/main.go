package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"activity-log-service/internal/infrastructure/metrics"
	"activity-log-service/internal/initialization"
	"activity-log-service/internal/server"
)

func main() {
	// Get configuration path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	// Initialize all dependencies
	deps, err := initialization.GetGRPCDependencies(configPath)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize dependencies")
	}
	defer func() {
		if err := deps.Cleanup(); err != nil {
			deps.Logger.WithError(err).Error("Failed to cleanup dependencies")
		}
	}()

	deps.Logger.Info("Starting gRPC server...")

	// Start metrics server
	metrics.StartMetricsServer(deps.Config.Metrics.Port, deps.Logger)

	// Create gRPC server
	grpcServer, err := server.NewGRPCServer(deps.UseCase, deps.Config, deps.Logger, deps.Tracer)
	if err != nil {
		deps.Logger.WithError(err).Fatal("Failed to create gRPC server")
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		deps.Logger.Info("Shutting down gRPC server...")
		cancel()
	}()

	// Start gRPC server
	deps.Logger.WithField("port", deps.Config.Server.GRPCPort).Info("gRPC server started")
	if err := grpcServer.Start(ctx); err != nil {
		deps.Logger.WithError(err).Fatal("gRPC server failed")
	}

	deps.Logger.Info("gRPC server shutdown complete")
}
