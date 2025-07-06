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
	deps, err := initialization.GetCronDependencies(configPath)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize dependencies")
	}
	defer func() {
		if err := deps.Cleanup(); err != nil {
			deps.Logger.WithError(err).Error("Failed to cleanup dependencies")
		}
	}()

	deps.Logger.Info("Starting cron server...")

	// Check if cron is enabled
	if !deps.Config.Cron.Enabled {
		deps.Logger.Info("Cron server is disabled in configuration")
		return
	}

	// Start metrics server (on different port for cron service)
	metricsPort := deps.Config.Metrics.Port + 3
	metrics.StartMetricsServer(metricsPort, deps.Logger)

	// Create cron server
	cronServer := server.NewCronServer(deps.Repository, deps.Cache, deps.Mailer, deps.Config, deps.Logger, deps.Tracer)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		deps.Logger.Info("Shutting down cron server...")
		cancel()
	}()

	// Start cron server
	deps.Logger.WithFields(logrus.Fields{
		"daily_summary_time": deps.Config.Cron.DailySummaryTime,
		"cleanup_interval":   deps.Config.Cron.CleanupInterval,
	}).Info("Cron server started")

	if err := cronServer.Start(ctx); err != nil {
		deps.Logger.WithError(err).Fatal("Cron server failed")
	}

	deps.Logger.Info("Cron server shutdown complete")
}
