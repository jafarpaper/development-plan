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

type MockActivityLogRepository struct {
	mock.Mock
}

func (m *MockActivityLogRepository) Create(ctx context.Context, activityLog *entity.ActivityLog) error {
	args := m.Called(ctx, activityLog)
	return args.Error(0)
}

func (m *MockActivityLogRepository) GetByID(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entity.ActivityLog), args.Error(1)
}

func (m *MockActivityLogRepository) GetByCompanyID(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockActivityLogRepository) Update(ctx context.Context, activityLog *entity.ActivityLog) error {
	args := m.Called(ctx, activityLog)
	return args.Error(0)
}

func (m *MockActivityLogRepository) Delete(ctx context.Context, id valueobject.ActivityLogID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockActivityLogRepository) GetByObjectID(ctx context.Context, companyID, objectID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, objectID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockActivityLogRepository) GetByActivityName(ctx context.Context, companyID, activityName string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, activityName, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockActivityLogRepository) GetByDateRange(ctx context.Context, companyID string, startDate, endDate time.Time, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, startDate, endDate, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockActivityLogRepository) GetByActor(ctx context.Context, companyID, actorID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, actorID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockActivityLogRepository) CountByCompanyID(ctx context.Context, companyID string) (int, error) {
	args := m.Called(ctx, companyID)
	return args.Int(0), args.Error(1)
}

type MockNATSPublisher struct {
	mock.Mock
}

func (m *MockNATSPublisher) PublishActivityLogCreated(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockNATSPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockNATSPublisher) EnsureStream(streamName, subject string) error {
	args := m.Called(streamName, subject)
	return args.Error(0)
}

type MockMailer struct {
	mock.Mock
}

func (m *MockMailer) SendActivityLogNotification(ctx context.Context, data interface{}) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockMailer) SendDailySummary(ctx context.Context, recipients []string, summaryData map[string]interface{}) error {
	args := m.Called(ctx, recipients, summaryData)
	return args.Error(0)
}

func (m *MockMailer) TestConnection(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestActivityLogUseCase_CreateActivityLog(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)
	mockPublisher := new(MockNATSPublisher)

	useCase := NewActivityLogUseCase(mockArangoRepo, mockPublisher, nil)

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

	mockArangoRepo.On("Create", ctx, mock.AnythingOfType("*entity.ActivityLog")).Return(nil)
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
	assert.Equal(t, req.ActorID, activityLog.ActorID)
	assert.Equal(t, req.ActorName, activityLog.ActorName)
	assert.Equal(t, req.ActorEmail, activityLog.ActorEmail)

	mockArangoRepo.AssertExpectations(t)
	mockPublisher.AssertExpectations(t)
}

func TestActivityLogUseCase_CreateActivityLog_InvalidActor(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)

	useCase := NewActivityLogUseCase(mockArangoRepo, nil, nil)

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
	assert.Contains(t, err.Error(), "invalid activity log")
}

func TestActivityLogUseCase_CreateActivityLog_InvalidJSON(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)

	useCase := NewActivityLogUseCase(mockArangoRepo, nil, nil)

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

func TestActivityLogUseCase_CreateActivityLog_ArangoError(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)

	useCase := NewActivityLogUseCase(mockArangoRepo, nil, nil)

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

	mockArangoRepo.On("Create", ctx, mock.AnythingOfType("*entity.ActivityLog")).Return(errors.New("arango error"))

	activityLog, err := useCase.CreateActivityLog(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, activityLog)
	assert.Contains(t, err.Error(), "failed to create activity log")
	mockArangoRepo.AssertExpectations(t)
}

func TestActivityLogUseCase_GetActivityLog(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)

	useCase := NewActivityLogUseCase(mockArangoRepo, nil, nil)

	ctx := context.Background()
	id := "valid-id"
	expectedLog := &entity.ActivityLog{
		ID:           valueobject.ActivityLogID(id),
		ActivityName: "user_created",
		CompanyID:    "company1",
	}

	mockArangoRepo.On("GetByID", ctx, valueobject.ActivityLogID(id)).Return(expectedLog, nil)

	activityLog, err := useCase.GetActivityLog(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, expectedLog, activityLog)
	mockArangoRepo.AssertExpectations(t)
}

func TestActivityLogUseCase_GetActivityLog_NotFound(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)

	useCase := NewActivityLogUseCase(mockArangoRepo, nil, nil)

	ctx := context.Background()
	id := "valid-id"

	mockArangoRepo.On("GetByID", ctx, valueobject.ActivityLogID(id)).Return((*entity.ActivityLog)(nil), entity.ErrActivityLogNotFound)

	activityLog, err := useCase.GetActivityLog(ctx, id)

	assert.Error(t, err)
	assert.Nil(t, activityLog)
	assert.Contains(t, err.Error(), "failed to get activity log")
	mockArangoRepo.AssertExpectations(t)
}

func TestActivityLogUseCase_ListActivityLogs(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)

	useCase := NewActivityLogUseCase(mockArangoRepo, nil, nil)

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

	mockArangoRepo.On("GetByCompanyID", ctx, companyID, page, limit).Return(expectedLogs, expectedTotal, nil)

	logs, total, err := useCase.ListActivityLogs(ctx, companyID, page, limit)

	require.NoError(t, err)
	assert.Equal(t, expectedLogs, logs)
	assert.Equal(t, expectedTotal, total)
	mockArangoRepo.AssertExpectations(t)
}

func TestActivityLogUseCase_ListActivityLogs_EmptyCompanyID(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)

	useCase := NewActivityLogUseCase(mockArangoRepo, nil, nil)

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

func TestActivityLogUseCase_ListActivityLogs_DefaultPagination(t *testing.T) {
	mockArangoRepo := new(MockActivityLogRepository)

	useCase := NewActivityLogUseCase(mockArangoRepo, nil, nil)

	ctx := context.Background()
	companyID := "company1"
	page := 0
	limit := 0

	expectedLogs := []*entity.ActivityLog{}
	expectedTotal := 0

	mockArangoRepo.On("GetByCompanyID", ctx, companyID, 1, 10).Return(expectedLogs, expectedTotal, nil)

	logs, total, err := useCase.ListActivityLogs(ctx, companyID, page, limit)

	require.NoError(t, err)
	assert.Equal(t, expectedLogs, logs)
	assert.Equal(t, expectedTotal, total)
	mockArangoRepo.AssertExpectations(t)
}
