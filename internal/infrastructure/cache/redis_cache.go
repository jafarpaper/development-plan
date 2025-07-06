package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type RedisCache struct {
	client *redis.Client
	logger *logrus.Logger
}

type CacheConfig struct {
	Address  string
	Password string
	DB       int
}

func NewRedisCache(config CacheConfig, logger *logrus.Logger) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
	})

	return &RedisCache{
		client: client,
		logger: logger,
	}
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for cache key %s: %w", key, err)
	}

	if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"key":        key,
			"expiration": expiration,
		}).Error("Failed to set cache value")
		return fmt.Errorf("failed to set cache value for key %s: %w", key, err)
	}

	c.logger.WithFields(logrus.Fields{
		"key":        key,
		"expiration": expiration,
	}).Debug("Cache value set successfully")

	return nil
}

func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache miss for key %s", key)
		}
		c.logger.WithError(err).WithField("key", key).Error("Failed to get cache value")
		return fmt.Errorf("failed to get cache value for key %s: %w", key, err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to unmarshal cache value")
		return fmt.Errorf("failed to unmarshal cache value for key %s: %w", key, err)
	}

	c.logger.WithField("key", key).Debug("Cache hit")
	return nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to delete cache value")
		return fmt.Errorf("failed to delete cache value for key %s: %w", key, err)
	}

	c.logger.WithField("key", key).Debug("Cache value deleted successfully")
	return nil
}

func (c *RedisCache) DeleteByPattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		c.logger.WithError(err).WithField("pattern", pattern).Error("Failed to get keys by pattern")
		return fmt.Errorf("failed to get keys by pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		c.logger.WithField("pattern", pattern).Debug("No keys found for pattern")
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"pattern": pattern,
			"keys":    keys,
		}).Error("Failed to delete keys by pattern")
		return fmt.Errorf("failed to delete keys by pattern %s: %w", pattern, err)
	}

	c.logger.WithFields(logrus.Fields{
		"pattern":    pattern,
		"keys_count": len(keys),
	}).Debug("Keys deleted successfully by pattern")

	return nil
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to check if key exists")
		return false, fmt.Errorf("failed to check if key exists %s: %w", key, err)
	}

	return count > 0, nil
}

func (c *RedisCache) SetExpiration(ctx context.Context, key string, expiration time.Duration) error {
	if err := c.client.Expire(ctx, key, expiration).Err(); err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"key":        key,
			"expiration": expiration,
		}).Error("Failed to set key expiration")
		return fmt.Errorf("failed to set expiration for key %s: %w", key, err)
	}

	c.logger.WithFields(logrus.Fields{
		"key":        key,
		"expiration": expiration,
	}).Debug("Key expiration set successfully")

	return nil
}

func (c *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Error("Failed to get key TTL")
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}

	return ttl, nil
}

func (c *RedisCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		c.logger.WithError(err).Error("Redis ping failed")
		return fmt.Errorf("redis ping failed: %w", err)
	}

	c.logger.Debug("Redis ping successful")
	return nil
}

func (c *RedisCache) Close() error {
	if err := c.client.Close(); err != nil {
		c.logger.WithError(err).Error("Failed to close Redis client")
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	c.logger.Info("Redis client closed successfully")
	return nil
}

func (c *RedisCache) FlushAll(ctx context.Context) error {
	if err := c.client.FlushAll(ctx).Err(); err != nil {
		c.logger.WithError(err).Error("Failed to flush all Redis keys")
		return fmt.Errorf("failed to flush all Redis keys: %w", err)
	}

	c.logger.Info("All Redis keys flushed successfully")
	return nil
}

// Cache key builders
func BuildActivityLogCacheKey(id string) string {
	return fmt.Sprintf("activity_log:%s", id)
}

func BuildCompanyActivityLogsCacheKey(companyID string, page, limit int) string {
	return fmt.Sprintf("company_activity_logs:%s:page:%d:limit:%d", companyID, page, limit)
}

func BuildActivityLogCountCacheKey(companyID string) string {
	return fmt.Sprintf("activity_log_count:%s", companyID)
}
