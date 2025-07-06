package server

import (
	"context"
	"fmt"
	"net"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"activity-log-service/internal/application/usecase"
	deliveryGRPC "activity-log-service/internal/delivery/grpc"
	"activity-log-service/internal/infrastructure/config"
	pb "activity-log-service/pkg/proto"
)

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	useCase  *usecase.ActivityLogUseCase
	config   *config.Config
	logger   *logrus.Logger
	tracer   opentracing.Tracer
}

func NewGRPCServer(
	useCase *usecase.ActivityLogUseCase,
	config *config.Config,
	logger *logrus.Logger,
	tracer opentracing.Tracer,
) (*GRPCServer, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Server.GRPCPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on gRPC port: %w", err)
	}

	server := grpc.NewServer()
	activityLogService := deliveryGRPC.NewActivityLogServiceServer(useCase, tracer)

	pb.RegisterActivityLogServiceServer(server, activityLogService)
	reflection.Register(server)

	return &GRPCServer{
		server:   server,
		listener: lis,
		useCase:  useCase,
		config:   config,
		logger:   logger,
		tracer:   tracer,
	}, nil
}

func (s *GRPCServer) Start(ctx context.Context) error {
	s.logger.WithField("port", s.config.Server.GRPCPort).Info("Starting gRPC server")

	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down gRPC server")
		s.server.GracefulStop()
	}()

	if err := s.server.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

func (s *GRPCServer) Stop() {
	s.logger.Info("Stopping gRPC server")
	s.server.GracefulStop()
}
