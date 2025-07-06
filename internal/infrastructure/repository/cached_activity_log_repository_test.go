package repository

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/valueobject"
)

// Mock repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, activityLog *entity.ActivityLog) error {
	args := m.Called(ctx, activityLog)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ActivityLog), args.Error(1)
}

func (m *MockRepository) GetByCompanyID(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockRepository) Update(ctx context.Context, activityLog *entity.ActivityLog) error {
	args := m.Called(ctx, activityLog)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id valueobject.ActivityLogID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) GetByObjectID(ctx context.Context, companyID, objectID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, objectID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetByActivityName(ctx context.Context, companyID, activityName string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, activityName, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetByDateRange(ctx context.Context, companyID string, startDate, endDate time.Time, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, startDate, endDate, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetByActor(ctx context.Context, companyID, actorID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	args := m.Called(ctx, companyID, actorID, page, limit)
	return args.Get(0).([]*entity.ActivityLog), args.Int(1), args.Error(2)
}

func (m *MockRepository) CountByCompanyID(ctx context.Context, companyID string) (int, error) {
	args := m.Called(ctx, companyID)
	return args.Int(0), args.Error(1)
}

// Mock cache
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockCache) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCache) DeleteByPattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCache) SetExpiration(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *MockCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockCache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCache) FlushAll(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewCachedActivityLogRepository(t *testing.T) {
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	logger := logrus.New()

	cachedRepo := NewCachedActivityLogRepository(mockRepo, mockCache, logger)

	assert.NotNil(t, cachedRepo)
	assert.Equal(t, mockRepo, cachedRepo.repo)
	assert.Equal(t, mockCache, cachedRepo.cache)
	assert.Equal(t, logger, cachedRepo.logger)
}

func TestCachedActivityLogRepository_Create(t *testing.T) {
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	logger := logrus.New()

	cachedRepo := NewCachedActivityLogRepository(mockRepo, mockCache, logger)

	ctx := context.Background()
	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	activityLog := entity.NewActivityLog(
		"user_created",
		"company1",
		"user",
		"user123",
		nil,
		"User was created",
		actor,
	)

	// Mock expectations
	mockRepo.On("Create", ctx, activityLog).Return(nil)
	mockCache.On("Set", ctx, mock.AnythingOfType("string"), activityLog, mock.AnythingOfType("time.Duration")).Return(nil)
	mockCache.On("DeleteByPattern", ctx, mock.AnythingOfType("string")).Return(nil)
	mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)

	err = cachedRepo.Create(ctx, activityLog)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCachedActivityLogRepository_GetByID_CacheHit(t *testing.T) {
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	logger := logrus.New()

	cachedRepo := NewCachedActivityLogRepository(mockRepo, mockCache, logger)

	ctx := context.Background()
	id := valueobject.NewActivityLogID()

	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	expectedLog := entity.NewActivityLog(
		"user_created",
		"company1",
		"user",
		"user123",
		nil,
		"User was created",
		actor,
	)
	expectedLog.ID = id

	// Mock cache hit
	mockCache.On("Get", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*entity.ActivityLog")).Return(nil).Run(func(args mock.Arguments) {
		dest := args.Get(2).(*entity.ActivityLog)
		*dest = *expectedLog
	})

	result, err := cachedRepo.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, expectedLog.ID, result.ID)
	mockCache.AssertExpectations(t)
	// Repository should not be called on cache hit
	mockRepo.AssertNotCalled(t, "GetByID")
}

func TestCachedActivityLogRepository_GetByID_CacheMiss(t *testing.T) {
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	logger := logrus.New()

	cachedRepo := NewCachedActivityLogRepository(mockRepo, mockCache, logger)

	ctx := context.Background()
	id := valueobject.NewActivityLogID()

	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	expectedLog := entity.NewActivityLog(
		"user_created",
		"company1",
		"user",
		"user123",
		nil,
		"User was created",
		actor,
	)
	expectedLog.ID = id

	// Mock cache miss
	mockCache.On("Get", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*entity.ActivityLog")).Return(assert.AnError)

	// Mock repository call
	mockRepo.On("GetByID", ctx, id).Return(expectedLog, nil)

	// Mock cache set after retrieval
	mockCache.On("Set", ctx, mock.AnythingOfType("string"), expectedLog, mock.AnythingOfType("time.Duration")).Return(nil)

	result, err := cachedRepo.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, expectedLog, result)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCachedActivityLogRepository_GetByCompanyID_CacheHit(t *testing.T) {
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	logger := logrus.New()

	cachedRepo := NewCachedActivityLogRepository(mockRepo, mockCache, logger)

	ctx := context.Background()
	companyID := "company1"
	page := 1
	limit := 10

	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	expectedLogs := []*entity.ActivityLog{
		entity.NewActivityLog(
			"user_created",
			companyID,
			"user",
			"user123",
			nil,
			"User was created",
			actor,
		),
	}
	expectedTotal := 1

	// Mock cache hit
	mockCache.On("Get", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*struct { ActivityLogs []*entity.ActivityLog \"json:\\\"activity_logs\\\"\"; Total int \"json:\\\"total\\\"\" }")).Return(nil).Run(func(args mock.Arguments) {
		dest := args.Get(2)
		// Set the cached result
		if result, ok := dest.(*struct {
			ActivityLogs []*entity.ActivityLog `json:"activity_logs"`
			Total        int                   `json:"total"`
		}); ok {
			result.ActivityLogs = expectedLogs
			result.Total = expectedTotal
		}
	})

	logs, total, err := cachedRepo.GetByCompanyID(ctx, companyID, page, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedLogs, logs)
	assert.Equal(t, expectedTotal, total)
	mockCache.AssertExpectations(t)
	// Repository should not be called on cache hit
	mockRepo.AssertNotCalled(t, "GetByCompanyID")
}
