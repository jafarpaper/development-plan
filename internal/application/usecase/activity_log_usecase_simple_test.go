package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/valueobject"
)

// Simple mock that only implements the methods we need
type SimpleActivityLogRepository struct {
	mock.Mock
}

func (m *SimpleActivityLogRepository) Create(ctx context.Context, activityLog *entity.ActivityLog) error {
	args := m.Called(ctx, activityLog)
	return args.Error(0)
}

func (m *SimpleActivityLogRepository) GetByID(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ActivityLog), args.Error(1)
}

func (m *SimpleActivityLogRepository) GetByCompanyID(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *SimpleActivityLogRepository) Update(ctx context.Context, activityLog *entity.ActivityLog) error {
	args := m.Called(ctx, activityLog)
	return args.Error(0)
}

func (m *SimpleActivityLogRepository) Delete(ctx context.Context, id valueobject.ActivityLogID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *SimpleActivityLogRepository) GetByObjectID(ctx context.Context, companyID, objectID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, objectID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *SimpleActivityLogRepository) GetByActivityName(ctx context.Context, companyID, activityName string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, activityName, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *SimpleActivityLogRepository) GetByDateRange(ctx context.Context, companyID string, startDate, endDate time.Time, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, startDate, endDate, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *SimpleActivityLogRepository) GetByActor(ctx context.Context, companyID, actorID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, actorID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *SimpleActivityLogRepository) CountByCompanyID(ctx context.Context, companyID string) (int, error) {
	args := m.Called(ctx, companyID)
	return args.Int(0), args.Error(1)
}

type SimplePublisher struct {
	mock.Mock
}

func (m *SimplePublisher) PublishActivityLogCreated(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *SimplePublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *SimplePublisher) EnsureStream(streamName, subject string) error {
	args := m.Called(streamName, subject)
	return args.Error(0)
}

func TestSimpleActivityLogUseCase_CreateActivityLog(t *testing.T) {
	mockRepo := new(SimpleActivityLogRepository)
	mockPublisher := new(SimplePublisher)

	useCase := NewActivityLogUseCase(mockRepo, mockPublisher, nil)

	ctx := context.Background()
	req := &CreateActivityLogRequest{
		ActivityName:     "user_created",
		CompanyID:        "company1",
		ObjectName:       "user",
		ObjectID:         "user123",
		Changes:          `{"field": "value"}`,
		FormattedMessage: "User was created",
		ActorID:          "actor1",
		ActorName:        "John Doe",
		ActorEmail:       "john@example.com",
	}

	mockRepo.On("Create", ctx, mock.AnythingOfType("*entity.ActivityLog")).Return(nil)
	mockPublisher.On("PublishActivityLogCreated", ctx, mock.Anything).Return(nil)

	activityLog, err := useCase.CreateActivityLog(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, activityLog)
	assert.Equal(t, req.ActivityName, activityLog.ActivityName)
	assert.Equal(t, req.CompanyID, activityLog.CompanyID)
	assert.Equal(t, req.ObjectName, activityLog.ObjectName)
	assert.Equal(t, req.ObjectID, activityLog.ObjectID)
	assert.Equal(t, json.RawMessage(req.Changes), activityLog.Changes)
	assert.Equal(t, req.FormattedMessage, activityLog.FormattedMessage)
	assert.Equal(t, req.ActorID, activityLog.Actor.ID)
	assert.Equal(t, req.ActorName, activityLog.Actor.Name)
	assert.Equal(t, req.ActorEmail, activityLog.Actor.Email)

	mockRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestSimpleActivityLogUseCase_CreateActivityLog_InvalidActor(t *testing.T) {
	mockRepo := new(SimpleActivityLogRepository)

	useCase := NewActivityLogUseCase(mockRepo, nil, nil)

	ctx := context.Background()
	req := &CreateActivityLogRequest{
		ActivityName:     "user_created",
		CompanyID:        "company1",
		ObjectName:       "user",
		ObjectID:         "user123",
		Changes:          `{"field": "value"}`,
		FormattedMessage: "User was created",
		ActorID:          "actor1",
		ActorName:        "John Doe",
		ActorEmail:       "invalid-email",
	}

	activityLog, err := useCase.CreateActivityLog(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, activityLog)
	assert.Contains(t, err.Error(), "invalid actor")
}

func TestSimpleActivityLogUseCase_CreateActivityLog_InvalidJSON(t *testing.T) {
	mockRepo := new(SimpleActivityLogRepository)

	useCase := NewActivityLogUseCase(mockRepo, nil, nil)

	ctx := context.Background()
	req := &CreateActivityLogRequest{
		ActivityName:     "user_created",
		CompanyID:        "company1",
		ObjectName:       "user",
		ObjectID:         "user123",
		Changes:          `{"field": "value"`,
		FormattedMessage: "User was created",
		ActorID:          "actor1",
		ActorName:        "John Doe",
		ActorEmail:       "john@example.com",
	}

	activityLog, err := useCase.CreateActivityLog(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, activityLog)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestSimpleActivityLogUseCase_GetActivityLog(t *testing.T) {
	mockRepo := new(SimpleActivityLogRepository)

	useCase := NewActivityLogUseCase(mockRepo, nil, nil)

	ctx := context.Background()
	id := "valid-id"
	expectedLog := &entity.ActivityLog{
		ID:           valueobject.ActivityLogID(id),
		ActivityName: "user_created",
		CompanyID:    "company1",
	}

	mockRepo.On("GetByID", ctx, valueobject.ActivityLogID(id)).Return(expectedLog, nil)

	activityLog, err := useCase.GetActivityLog(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, expectedLog, activityLog)
	mockRepo.AssertExpectations(t)
}

func TestSimpleActivityLogUseCase_ListActivityLogs(t *testing.T) {
	mockRepo := new(SimpleActivityLogRepository)

	useCase := NewActivityLogUseCase(mockRepo, nil, nil)

	ctx := context.Background()
	companyID := "company1"
	page := 1
	limit := 10

	expectedLogs := []*entity.ActivityLog{
		{
			ID:           valueobject.NewActivityLogID(),
			ActivityName: "user_created",
			CompanyID:    companyID,
		},
	}
	expectedTotal := 1

	mockRepo.On("GetByCompanyID", ctx, companyID, page, limit).Return(expectedLogs, expectedTotal, nil)

	logs, total, err := useCase.ListActivityLogs(ctx, companyID, page, limit)

	require.NoError(t, err)
	assert.Equal(t, expectedLogs, logs)
	assert.Equal(t, expectedTotal, total)
	mockRepo.AssertExpectations(t)
}

func TestSimpleActivityLogUseCase_ListActivityLogs_EmptyCompanyID(t *testing.T) {
	mockRepo := new(SimpleActivityLogRepository)

	useCase := NewActivityLogUseCase(mockRepo, nil, nil)

	ctx := context.Background()
	companyID := ""
	page := 1
	limit := 10

	logs, total, err := useCase.ListActivityLogs(ctx, companyID, page, limit)

	assert.Error(t, err)
	assert.Nil(t, logs)
	assert.Equal(t, 0, total)
	assert.Contains(t, err.Error(), "company ID is required")
}
