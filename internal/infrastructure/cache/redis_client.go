package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/valueobject"
)

type RedisClient struct {
	client *redis.Client
	logger *logrus.Logger
}

func NewRedisClient(addr, password string, db int, logger *logrus.Logger) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisClient{
		client: rdb,
		logger: logger,
	}
}

func (r *RedisClient) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}
	return nil
}

func (r *RedisClient) SetActivityLog(ctx context.Context, activityLog *entity.ActivityLog, expiration time.Duration) error {
	key := r.getActivityLogKey(activityLog.ID)

	data, err := json.Marshal(activityLog)
	if err != nil {
		return fmt.Errorf("failed to marshal activity log: %w", err)
	}

	err = r.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set activity log in cache: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"activity_log_id": activityLog.ID.String(),
		"expiration":      expiration,
	}).Debug("Activity log cached")

	return nil
}

func (r *RedisClient) GetActivityLog(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error) {
	key := r.getActivityLogKey(id)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("activity log not found in cache")
		}
		return nil, fmt.Errorf("failed to get activity log from cache: %w", err)
	}

	var activityLog entity.ActivityLog
	err = json.Unmarshal([]byte(data), &activityLog)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal activity log: %w", err)
	}

	r.logger.WithField("activity_log_id", id.String()).Debug("Activity log retrieved from cache")

	return &activityLog, nil
}

func (r *RedisClient) DeleteActivityLog(ctx context.Context, id valueobject.ActivityLogID) error {
	key := r.getActivityLogKey(id)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete activity log from cache: %w", err)
	}

	r.logger.WithField("activity_log_id", id.String()).Debug("Activity log deleted from cache")

	return nil
}

func (r *RedisClient) SetCompanyActivityLogs(ctx context.Context, companyID string, page, limit int, logs []*entity.ActivityLog, expiration time.Duration) error {
	key := r.getCompanyLogsKey(companyID, page, limit)

	data, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal activity logs: %w", err)
	}

	err = r.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set company activity logs in cache: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"company_id": companyID,
		"page":       page,
		"limit":      limit,
		"count":      len(logs),
		"expiration": expiration,
	}).Debug("Company activity logs cached")

	return nil
}

func (r *RedisClient) GetCompanyActivityLogs(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, error) {
	key := r.getCompanyLogsKey(companyID, page, limit)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("company activity logs not found in cache")
		}
		return nil, fmt.Errorf("failed to get company activity logs from cache: %w", err)
	}

	var logs []*entity.ActivityLog
	err = json.Unmarshal([]byte(data), &logs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal activity logs: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"company_id": companyID,
		"page":       page,
		"limit":      limit,
		"count":      len(logs),
	}).Debug("Company activity logs retrieved from cache")

	return logs, nil
}

func (r *RedisClient) InvalidateCompanyCache(ctx context.Context, companyID string) error {
	pattern := fmt.Sprintf("activity_logs:company:%s:*", companyID)

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}

	if len(keys) > 0 {
		err = r.client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete company cache keys: %w", err)
		}

		r.logger.WithFields(logrus.Fields{
			"company_id":   companyID,
			"deleted_keys": len(keys),
		}).Debug("Company cache invalidated")
	}

	return nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) getActivityLogKey(id valueobject.ActivityLogID) string {
	return fmt.Sprintf("activity_log:%s", id.String())
}

func (r *RedisClient) getCompanyLogsKey(companyID string, page, limit int) string {
	return fmt.Sprintf("activity_logs:company:%s:page:%d:limit:%d", companyID, page, limit)
}

type CacheRepository interface {
	SetActivityLog(ctx context.Context, activityLog *entity.ActivityLog, expiration time.Duration) error
	GetActivityLog(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error)
	DeleteActivityLog(ctx context.Context, id valueobject.ActivityLogID) error
	SetCompanyActivityLogs(ctx context.Context, companyID string, page, limit int, logs []*entity.ActivityLog, expiration time.Duration) error
	GetCompanyActivityLogs(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, error)
	InvalidateCompanyCache(ctx context.Context, companyID string) error
	Ping(ctx context.Context) error
	Close() error
}
