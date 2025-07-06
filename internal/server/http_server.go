package server

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"activity-log-service/internal/application/usecase"
	"activity-log-service/internal/delivery/http"
	"activity-log-service/internal/infrastructure/config"
)

type HTTPServer struct {
	echoServer *http.EchoServer
	useCase    *usecase.ActivityLogUseCase
	config     *config.Config
	logger     *logrus.Logger
	tracer     opentracing.Tracer
}

func NewHTTPServer(
	useCase *usecase.ActivityLogUseCase,
	config *config.Config,
	logger *logrus.Logger,
	tracer opentracing.Tracer,
) *HTTPServer {
	echoServer := http.NewEchoServer(useCase, tracer)

	return &HTTPServer{
		echoServer: echoServer,
		useCase:    useCase,
		config:     config,
		logger:     logger,
		tracer:     tracer,
	}
}

func (s *HTTPServer) Start(ctx context.Context) error {
	address := fmt.Sprintf(":%d", s.config.Server.Port)
	s.logger.WithField("port", s.config.Server.Port).Info("Starting HTTP server")

	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down HTTP server")
		if err := s.echoServer.Shutdown(ctx); err != nil {
			s.logger.WithError(err).Error("Failed to shutdown HTTP server gracefully")
		}
	}()

	if err := s.echoServer.Start(address); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	return s.echoServer.Shutdown(ctx)
}
