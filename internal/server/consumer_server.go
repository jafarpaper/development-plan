package server

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"activity-log-service/internal/domain/repository"
	"activity-log-service/internal/infrastructure/config"
	"activity-log-service/internal/infrastructure/messaging"
)

type ConsumerServer struct {
	consumer   *messaging.NATSConsumer
	arangoRepo repository.ActivityLogRepository
	config     *config.Config
	logger     *logrus.Logger
	tracer     opentracing.Tracer
}

func NewConsumerServer(
	arangoRepo repository.ActivityLogRepository,
	config *config.Config,
	logger *logrus.Logger,
	tracer opentracing.Tracer,
) (*ConsumerServer, error) {
	consumer, err := messaging.NewNATSConsumer(
		config.NATS.URL,
		logger,
		arangoRepo,
		4, // Number of workers
		tracer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS consumer: %w", err)
	}

	return &ConsumerServer{
		consumer:   consumer,
		arangoRepo: arangoRepo,
		config:     config,
		logger:     logger,
		tracer:     tracer,
	}, nil
}

func (s *ConsumerServer) Start(ctx context.Context) error {
	s.logger.WithField("url", s.config.NATS.URL).Info("Starting NATS consumer")

	if err := s.consumer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start NATS consumer: %w", err)
	}

	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down NATS consumer")
		s.consumer.Stop()
	}()

	// Wait for consumer to finish
	s.consumer.Wait()
	return nil
}

func (s *ConsumerServer) Stop() {
	s.logger.Info("Stopping NATS consumer")
	s.consumer.Stop()
}
