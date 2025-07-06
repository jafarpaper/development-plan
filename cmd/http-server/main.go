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
	deps, err := initialization.GetHTTPDependencies(configPath)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize dependencies")
	}
	defer func() {
		if err := deps.Cleanup(); err != nil {
			deps.Logger.WithError(err).Error("Failed to cleanup dependencies")
		}
	}()

	deps.Logger.Info("Starting HTTP server...")

	// Start metrics server (on different port for HTTP service)
	metricsPort := deps.Config.Metrics.Port + 1
	metrics.StartMetricsServer(metricsPort, deps.Logger)

	// Create HTTP server
	httpServer := server.NewHTTPServer(deps.UseCase, deps.Config, deps.Logger, deps.Tracer)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		deps.Logger.Info("Shutting down HTTP server...")
		cancel()
	}()

	// Start HTTP server
	deps.Logger.WithField("port", deps.Config.Server.Port).Info("HTTP server started")
	if err := httpServer.Start(ctx); err != nil {
		deps.Logger.WithError(err).Fatal("HTTP server failed")
	}

	deps.Logger.Info("HTTP server shutdown complete")
}
