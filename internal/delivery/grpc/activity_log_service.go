package grpc

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"activity-log-service/internal/application/usecase"
	"activity-log-service/internal/domain/entity"
	pb "activity-log-service/pkg/proto"
)

type ActivityLogServiceServer struct {
	pb.UnimplementedActivityLogServiceServer
	useCase *usecase.ActivityLogUseCase
	tracer  opentracing.Tracer
}

func NewActivityLogServiceServer(useCase *usecase.ActivityLogUseCase, tracer opentracing.Tracer) *ActivityLogServiceServer {
	return &ActivityLogServiceServer{
		useCase: useCase,
		tracer:  tracer,
	}
}

func (s *ActivityLogServiceServer) CreateActivityLog(ctx context.Context, req *pb.CreateActivityLogRequest) (*pb.CreateActivityLogResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CreateActivityLog")
	defer span.Finish()

	ext.Component.Set(span, "grpc")
	span.SetTag("activity_name", req.ActivityName)
	span.SetTag("company_id", req.CompanyId)
	if req.ActivityName == "" {
		return nil, status.Error(codes.InvalidArgument, "activity name is required")
	}
	if req.CompanyId == "" {
		return nil, status.Error(codes.InvalidArgument, "company ID is required")
	}
	if req.ObjectName == "" {
		return nil, status.Error(codes.InvalidArgument, "object name is required")
	}
	if req.ObjectId == "" {
		return nil, status.Error(codes.InvalidArgument, "object ID is required")
	}
	if req.FormattedMessage == "" {
		return nil, status.Error(codes.InvalidArgument, "formatted message is required")
	}
	if req.ActorId == "" {
		return nil, status.Error(codes.InvalidArgument, "actor ID is required")
	}
	if req.ActorName == "" {
		return nil, status.Error(codes.InvalidArgument, "actor name is required")
	}
	if req.ActorEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "actor email is required")
	}

	useCaseReq := &usecase.CreateActivityLogRequest{
		ActivityName:     req.ActivityName,
		CompanyID:        req.CompanyId,
		ObjectName:       req.ObjectName,
		ObjectID:         req.ObjectId,
		Changes:          req.Changes,
		FormattedMessage: req.FormattedMessage,
		ActorID:          req.ActorId,
		ActorName:        req.ActorName,
		ActorEmail:       req.ActorEmail,
	}

	activityLog, err := s.useCase.CreateActivityLog(ctx, useCaseReq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create activity log: %v", err))
	}

	return &pb.CreateActivityLogResponse{
		ActivityLog: s.entityToProto(activityLog),
	}, nil
}

func (s *ActivityLogServiceServer) GetActivityLog(ctx context.Context, req *pb.GetActivityLogRequest) (*pb.GetActivityLogResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetActivityLog")
	defer span.Finish()

	ext.Component.Set(span, "grpc")
	span.SetTag("activity_log_id", req.Id)
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "activity log ID is required")
	}

	activityLog, err := s.useCase.GetActivityLog(ctx, req.Id)
	if err != nil {
		if err == entity.ErrActivityLogNotFound {
			return nil, status.Error(codes.NotFound, "activity log not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get activity log: %v", err))
	}

	return &pb.GetActivityLogResponse{
		ActivityLog: s.entityToProto(activityLog),
	}, nil
}

func (s *ActivityLogServiceServer) ListActivityLogs(ctx context.Context, req *pb.ListActivityLogsRequest) (*pb.ListActivityLogsResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ListActivityLogs")
	defer span.Finish()

	ext.Component.Set(span, "grpc")
	span.SetTag("company_id", req.CompanyId)
	span.SetTag("page", req.Page)
	span.SetTag("limit", req.Limit)
	if req.CompanyId == "" {
		return nil, status.Error(codes.InvalidArgument, "company ID is required")
	}

	page := int(req.Page)
	limit := int(req.Limit)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	activityLogs, total, err := s.useCase.ListActivityLogs(ctx, req.CompanyId, page, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list activity logs: %v", err))
	}

	protoLogs := make([]*pb.ActivityLog, len(activityLogs))
	for i, log := range activityLogs {
		protoLogs[i] = s.entityToProto(log)
	}

	return &pb.ListActivityLogsResponse{
		ActivityLogs: protoLogs,
		Total:        int32(total),
		Page:         int32(page),
		Limit:        int32(limit),
	}, nil
}

func (s *ActivityLogServiceServer) entityToProto(entity *entity.ActivityLog) *pb.ActivityLog {
	return &pb.ActivityLog{
		Id:               entity.ID.String(),
		ActivityName:     entity.ActivityName,
		CompanyId:        entity.CompanyID,
		ObjectName:       entity.ObjectName,
		ObjectId:         entity.ObjectID,
		Changes:          string(entity.Changes),
		FormattedMessage: entity.FormattedMessage,
		ActorId:          entity.ActorID,
		ActorName:        entity.ActorName,
		ActorEmail:       entity.ActorEmail,
		CreatedAt:        timestamppb.New(entity.CreatedAt),
	}
}
