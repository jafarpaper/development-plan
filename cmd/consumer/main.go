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
	deps, err := initialization.GetConsumerDependencies(configPath)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize dependencies")
	}
	defer func() {
		if err := deps.Cleanup(); err != nil {
			deps.Logger.WithError(err).Error("Failed to cleanup dependencies")
		}
	}()

	deps.Logger.Info("Starting NATS consumer...")

	// Start metrics server (on different port for consumer service)
	metricsPort := deps.Config.Metrics.Port + 2
	metrics.StartMetricsServer(metricsPort, deps.Logger)

	// Create NATS consumer server
	consumerServer, err := server.NewConsumerServer(deps.Repository, deps.Config, deps.Logger, deps.Tracer)
	if err != nil {
		deps.Logger.WithError(err).Fatal("Failed to create consumer server")
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		deps.Logger.Info("Shutting down NATS consumer...")
		cancel()
	}()

	// Start NATS consumer
	deps.Logger.WithFields(logrus.Fields{
		"stream":  deps.Config.NATS.Stream,
		"subject": deps.Config.NATS.Subject,
		"durable": deps.Config.NATS.Durable,
	}).Info("NATS consumer started")

	if err := consumerServer.Start(ctx); err != nil {
		deps.Logger.WithError(err).Fatal("NATS consumer failed")
	}

	deps.Logger.Info("NATS consumer shutdown complete")
}
