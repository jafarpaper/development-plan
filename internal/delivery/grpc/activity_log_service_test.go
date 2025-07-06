package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"activity-log-service/internal/application/usecase"
	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/valueobject"
	pb "activity-log-service/pkg/proto"
)

type MockActivityLogUseCase struct {
	mock.Mock
}

func (m *MockActivityLogUseCase) CreateActivityLog(ctx context.Context, req *usecase.CreateActivityLogRequest) (*entity.ActivityLog, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*entity.ActivityLog), args.Error(1)
}

func (m *MockActivityLogUseCase) GetActivityLog(ctx context.Context, id string) (*entity.ActivityLog, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entity.ActivityLog), args.Error(1)
}

func (m *MockActivityLogUseCase) ListActivityLogs(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func TestActivityLogServiceServer_CreateActivityLog(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()
	req := &pb.CreateActivityLogRequest{
		ActivityName:     "user_created",
		CompanyId:        "company1",
		ObjectName:       "user",
		ObjectId:         "user123",
		Changes:          `{"field": "value"}`,
		FormattedMessage: "User was created",
		ActorId:          "actor1",
		ActorName:        "John Doe",
		ActorEmail:       "john@example.com",
	}

	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	expectedLog := &entity.ActivityLog{
		ID:               valueobject.NewActivityLogID(),
		ActivityName:     "user_created",
		CompanyID:        "company1",
		ObjectName:       "user",
		ObjectID:         "user123",
		FormattedMessage: "User was created",
		Actor:            actor,
	}

	mockUseCase.On("CreateActivityLog", ctx, mock.AnythingOfType("*usecase.CreateActivityLogRequest")).Return(expectedLog, nil)

	resp, err := server.CreateActivityLog(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.ActivityLog)
	assert.Equal(t, expectedLog.ID.String(), resp.ActivityLog.Id)
	assert.Equal(t, expectedLog.ActivityName, resp.ActivityLog.ActivityName)
	assert.Equal(t, expectedLog.CompanyID, resp.ActivityLog.CompanyId)
	mockUseCase.AssertExpectations(t)
}

func TestActivityLogServiceServer_CreateActivityLog_ValidationErrors(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()

	tests := []struct {
		name    string
		req     *pb.CreateActivityLogRequest
		wantErr codes.Code
	}{
		{
			name: "empty activity name",
			req: &pb.CreateActivityLogRequest{
				ActivityName:     "",
				CompanyId:        "company1",
				ObjectName:       "user",
				ObjectId:         "user123",
				FormattedMessage: "User was created",
				ActorId:          "actor1",
				ActorName:        "John Doe",
				ActorEmail:       "john@example.com",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "empty company id",
			req: &pb.CreateActivityLogRequest{
				ActivityName:     "user_created",
				CompanyId:        "",
				ObjectName:       "user",
				ObjectId:         "user123",
				FormattedMessage: "User was created",
				ActorId:          "actor1",
				ActorName:        "John Doe",
				ActorEmail:       "john@example.com",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "empty actor email",
			req: &pb.CreateActivityLogRequest{
				ActivityName:     "user_created",
				CompanyId:        "company1",
				ObjectName:       "user",
				ObjectId:         "user123",
				FormattedMessage: "User was created",
				ActorId:          "actor1",
				ActorName:        "John Doe",
				ActorEmail:       "",
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.CreateActivityLog(ctx, tt.req)

			assert.Nil(t, resp)
			assert.Error(t, err)

			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantErr, st.Code())
		})
	}
}

func TestActivityLogServiceServer_CreateActivityLog_UseCaseError(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()
	req := &pb.CreateActivityLogRequest{
		ActivityName:     "user_created",
		CompanyId:        "company1",
		ObjectName:       "user",
		ObjectId:         "user123",
		Changes:          `{"field": "value"}`,
		FormattedMessage: "User was created",
		ActorId:          "actor1",
		ActorName:        "John Doe",
		ActorEmail:       "john@example.com",
	}

	mockUseCase.On("CreateActivityLog", ctx, mock.AnythingOfType("*usecase.CreateActivityLogRequest")).Return((*entity.ActivityLog)(nil), errors.New("use case error"))

	resp, err := server.CreateActivityLog(ctx, req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	mockUseCase.AssertExpectations(t)
}

func TestActivityLogServiceServer_GetActivityLog(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()
	req := &pb.GetActivityLogRequest{
		Id: "valid-id",
	}

	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	expectedLog := &entity.ActivityLog{
		ID:           valueobject.ActivityLogID("valid-id"),
		ActivityName: "user_created",
		CompanyID:    "company1",
		Actor:        actor,
	}

	mockUseCase.On("GetActivityLog", ctx, "valid-id").Return(expectedLog, nil)

	resp, err := server.GetActivityLog(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.ActivityLog)
	assert.Equal(t, expectedLog.ID.String(), resp.ActivityLog.Id)
	assert.Equal(t, expectedLog.ActivityName, resp.ActivityLog.ActivityName)
	mockUseCase.AssertExpectations(t)
}

func TestActivityLogServiceServer_GetActivityLog_EmptyID(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()
	req := &pb.GetActivityLogRequest{
		Id: "",
	}

	resp, err := server.GetActivityLog(ctx, req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityLogServiceServer_GetActivityLog_NotFound(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()
	req := &pb.GetActivityLogRequest{
		Id: "non-existent-id",
	}

	mockUseCase.On("GetActivityLog", ctx, "non-existent-id").Return((*entity.ActivityLog)(nil), entity.ErrActivityLogNotFound)

	resp, err := server.GetActivityLog(ctx, req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	mockUseCase.AssertExpectations(t)
}

func TestActivityLogServiceServer_ListActivityLogs(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()
	req := &pb.ListActivityLogsRequest{
		CompanyId: "company1",
		Page:      1,
		Limit:     10,
	}

	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	expectedLogs := []*entity.ActivityLog{
		{
			ID:           valueobject.NewActivityLogID(),
			ActivityName: "user_created",
			CompanyID:    "company1",
			Actor:        actor,
		},
	}
	expectedTotal := 1

	mockUseCase.On("ListActivityLogs", ctx, "company1", 1, 10).Return(expectedLogs, expectedTotal, nil)

	resp, err := server.ListActivityLogs(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.ActivityLogs, 1)
	assert.Equal(t, int32(expectedTotal), resp.Total)
	assert.Equal(t, int32(1), resp.Page)
	assert.Equal(t, int32(10), resp.Limit)
	mockUseCase.AssertExpectations(t)
}

func TestActivityLogServiceServer_ListActivityLogs_EmptyCompanyID(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()
	req := &pb.ListActivityLogsRequest{
		CompanyId: "",
		Page:      1,
		Limit:     10,
	}

	resp, err := server.ListActivityLogs(ctx, req)

	assert.Nil(t, resp)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityLogServiceServer_ListActivityLogs_DefaultPagination(t *testing.T) {
	mockUseCase := new(MockActivityLogUseCase)
	server := NewActivityLogServiceServer(mockUseCase, nil)

	ctx := context.Background()
	req := &pb.ListActivityLogsRequest{
		CompanyId: "company1",
		Page:      0,
		Limit:     0,
	}

	expectedLogs := []*entity.ActivityLog{}
	expectedTotal := 0

	mockUseCase.On("ListActivityLogs", ctx, "company1", 1, 10).Return(expectedLogs, expectedTotal, nil)

	resp, err := server.ListActivityLogs(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.Page)
	assert.Equal(t, int32(10), resp.Limit)
	mockUseCase.AssertExpectations(t)
}
