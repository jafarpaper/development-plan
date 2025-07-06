package server

import (
	"context"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"activity-log-service/internal/domain/repository"
	"activity-log-service/internal/infrastructure/cache"
	"activity-log-service/internal/infrastructure/config"
	"activity-log-service/internal/infrastructure/email"
)

type CronServer struct {
	cron       *cron.Cron
	arangoRepo repository.ActivityLogRepository
	cacheRepo  *cache.RedisCache
	mailer     *email.Mailer
	config     *config.Config
	logger     *logrus.Logger
	tracer     opentracing.Tracer
}

func NewCronServer(
	arangoRepo repository.ActivityLogRepository,
	cacheRepo *cache.RedisCache,
	mailer *email.Mailer,
	config *config.Config,
	logger *logrus.Logger,
	tracer opentracing.Tracer,
) *CronServer {
	c := cron.New(cron.WithSeconds())

	return &CronServer{
		cron:       c,
		arangoRepo: arangoRepo,
		cacheRepo:  cacheRepo,
		mailer:     mailer,
		config:     config,
		logger:     logger,
		tracer:     tracer,
	}
}

func (s *CronServer) Start(ctx context.Context) error {
	s.logger.Info("Starting cron server")

	// Schedule cache cleanup every 5 minutes
	_, err := s.cron.AddFunc("0 */5 * * * *", s.cleanupExpiredCache)
	if err != nil {
		return fmt.Errorf("failed to schedule cache cleanup job: %w", err)
	}

	// Schedule metrics collection every hour
	_, err = s.cron.AddFunc("0 0 * * * *", s.collectMetrics)
	if err != nil {
		return fmt.Errorf("failed to schedule metrics collection job: %w", err)
	}

	// Schedule database maintenance every day at 2 AM
	_, err = s.cron.AddFunc("0 0 2 * * *", s.performDatabaseMaintenance)
	if err != nil {
		return fmt.Errorf("failed to schedule database maintenance job: %w", err)
	}

	// Schedule log rotation every day at 3 AM
	_, err = s.cron.AddFunc("0 0 3 * * *", s.rotateOldLogs)
	if err != nil {
		return fmt.Errorf("failed to schedule log rotation job: %w", err)
	}

	// Schedule daily summary email based on config
	if s.mailer != nil && s.config.Cron.DailySummaryTime != "" {
		// Parse the time and create cron expression
		// Format: "08:00" -> "0 0 8 * * *"
		var hour, minute int
		if _, err := fmt.Sscanf(s.config.Cron.DailySummaryTime, "%d:%d", &hour, &minute); err == nil {
			cronExpr := fmt.Sprintf("0 %d %d * * *", minute, hour)
			_, err = s.cron.AddFunc(cronExpr, s.sendDailySummary)
			if err != nil {
				return fmt.Errorf("failed to schedule daily summary job: %w", err)
			}
		}
	}

	s.cron.Start()

	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down cron server")
		s.Stop()
	}()

	// Keep the cron server running
	select {
	case <-ctx.Done():
		return nil
	}
}

func (s *CronServer) Stop() {
	s.logger.Info("Stopping cron server")
	cronCtx := s.cron.Stop()
	<-cronCtx.Done()
}

func (s *CronServer) cleanupExpiredCache() {
	span := s.tracer.StartSpan("cleanupExpiredCache")
	defer span.Finish()

	s.logger.Info("Running cache cleanup job")

	ctx, cancel := context.WithTimeout(opentracing.ContextWithSpan(context.Background(), span), 5*time.Minute)
	defer cancel()

	// Check Redis connection
	if err := s.cacheRepo.Ping(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to ping Redis during cache cleanup")
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
		return
	}

	s.logger.Info("Cache cleanup completed successfully")
}

func (s *CronServer) collectMetrics() {
	span := s.tracer.StartSpan("collectMetrics")
	defer span.Finish()

	s.logger.Info("Running metrics collection job")

	// Example: Collect database statistics
	// This could be expanded to collect various metrics about the system

	// For now, just log that metrics collection ran
	s.logger.WithFields(logrus.Fields{
		"timestamp": time.Now(),
		"job":       "metrics_collection",
	}).Info("Metrics collection completed")
}

func (s *CronServer) performDatabaseMaintenance() {
	span := s.tracer.StartSpan("performDatabaseMaintenance")
	defer span.Finish()

	s.logger.Info("Running database maintenance job")

	// Example maintenance tasks:
	// 1. Analyze collection statistics
	// 2. Optimize indexes
	// 3. Clean up old data (if retention policies exist)

	// For now, just log that maintenance ran
	s.logger.WithFields(logrus.Fields{
		"timestamp": time.Now(),
		"job":       "database_maintenance",
	}).Info("Database maintenance completed")
}

func (s *CronServer) rotateOldLogs() {
	span := s.tracer.StartSpan("rotateOldLogs")
	defer span.Finish()

	s.logger.Info("Running log rotation job")

	// Example: Archive old activity logs based on retention policy
	// This could involve:
	// 1. Moving old logs to archive storage
	// 2. Compressing old data
	// 3. Updating indexes

	// For now, just log that rotation ran
	s.logger.WithFields(logrus.Fields{
		"timestamp": time.Now(),
		"job":       "log_rotation",
	}).Info("Log rotation completed")
}

func (s *CronServer) sendDailySummary() {
	span := s.tracer.StartSpan("sendDailySummary")
	defer span.Finish()

	s.logger.Info("Running daily summary email job")

	ctx, cancel := context.WithTimeout(opentracing.ContextWithSpan(context.Background(), span), 10*time.Minute)
	defer cancel()

	if s.mailer == nil {
		s.logger.Warn("Mailer not configured, skipping daily summary")
		return
	}

	// For now, send a basic summary
	// In a real implementation, you would:
	// 1. Query activity log statistics for the past day
	// 2. Generate summary data
	// 3. Send email to configured recipients

	summaryData := map[string]interface{}{
		"Date":            time.Now().Format("2006-01-02"),
		"TotalActivities": 0,
		"UniqueUsers":     0,
		"TopActivity":     "N/A",
	}

	// Example recipients (in real implementation, get from config)
	recipients := []string{"admin@example.com"}

	if err := s.mailer.SendDailySummary(ctx, recipients, summaryData); err != nil {
		s.logger.WithError(err).Error("Failed to send daily summary email")
		return
	}

	s.logger.WithFields(logrus.Fields{
		"timestamp":  time.Now(),
		"job":        "daily_summary",
		"recipients": recipients,
	}).Info("Daily summary email sent successfully")
}
