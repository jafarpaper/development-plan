package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCacheKeys(t *testing.T) {
	// Test cache key builders
	assert.Equal(t, "activity_log:123", BuildActivityLogCacheKey("123"))
	assert.Equal(t, "company_activity_logs:company1:page:1:limit:10", BuildCompanyActivityLogsCacheKey("company1", 1, 10))
	assert.Equal(t, "activity_log_count:company1", BuildActivityLogCountCacheKey("company1"))
}

func TestNewRedisCache(t *testing.T) {
	logger := logrus.New()
	config := CacheConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}

	cache := NewRedisCache(config, logger)
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.client)
	assert.NotNil(t, cache.logger)
}

// Mock Redis client for testing without actual Redis server
type MockRedisClient struct {
	data map[string]string
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
	}
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	// In a real test, we'd use a proper mock library
	// For now, just store the value
	if str, ok := value.(string); ok {
		m.data[key] = str
	}
	return redis.NewStatusCmd(ctx)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	// Mock implementation
	cmd := redis.NewStringCmd(ctx)
	if value, exists := m.data[key]; exists {
		cmd.SetVal(value)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

// Integration test that would require actual Redis server
func TestRedisCache_Integration(t *testing.T) {
	// Skip this test if Redis is not available
	t.Skip("Skipping Redis integration test - requires running Redis server")

	logger := logrus.New()
	config := CacheConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}

	cache := NewRedisCache(config, logger)
	ctx := context.Background()

	// Test ping
	err := cache.Ping(ctx)
	if err != nil {
		t.Skip("Redis server not available, skipping integration test")
	}

	// Test set and get
	testKey := "test_key"
	testValue := map[string]interface{}{
		"id":   "123",
		"name": "test",
	}

	err = cache.Set(ctx, testKey, testValue, time.Minute)
	require.NoError(t, err)

	var result map[string]interface{}
	err = cache.Get(ctx, testKey, &result)
	require.NoError(t, err)
	assert.Equal(t, testValue, result)

	// Test exists
	exists, err := cache.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test delete
	err = cache.Delete(ctx, testKey)
	require.NoError(t, err)

	// Verify deletion
	exists, err = cache.Exists(ctx, testKey)
	require.NoError(t, err)
	assert.False(t, exists)
}
