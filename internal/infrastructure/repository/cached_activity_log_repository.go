package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/repository"
	"activity-log-service/internal/domain/valueobject"
	"activity-log-service/internal/infrastructure/cache"
)

type CachedActivityLogRepository struct {
	repo   repository.ActivityLogRepository
	cache  *cache.RedisCache
	logger *logrus.Logger
}

func NewCachedActivityLogRepository(
	repo repository.ActivityLogRepository,
	cache *cache.RedisCache,
	logger *logrus.Logger,
) *CachedActivityLogRepository {
	return &CachedActivityLogRepository{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (r *CachedActivityLogRepository) Create(ctx context.Context, activityLog *entity.ActivityLog) error {
	// First create in the main repository
	if err := r.repo.Create(ctx, activityLog); err != nil {
		return err
	}

	// Cache the created activity log
	cacheKey := cache.BuildActivityLogCacheKey(string(activityLog.ID))
	if err := r.cache.Set(ctx, cacheKey, activityLog, 1*time.Hour); err != nil {
		r.logger.WithError(err).WithField("activity_log_id", activityLog.ID).
			Warn("Failed to cache activity log after creation")
	}

	// Invalidate company activity logs cache
	if err := r.invalidateCompanyCache(ctx, activityLog.CompanyID); err != nil {
		r.logger.WithError(err).WithField("company_id", activityLog.CompanyID).
			Warn("Failed to invalidate company cache after creation")
	}

	return nil
}

func (r *CachedActivityLogRepository) GetByID(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error) {
	// Try to get from cache first
	cacheKey := cache.BuildActivityLogCacheKey(string(id))
	var activityLog entity.ActivityLog
	if err := r.cache.Get(ctx, cacheKey, &activityLog); err == nil {
		r.logger.WithField("activity_log_id", id).Debug("Activity log retrieved from cache")
		return &activityLog, nil
	}

	// If not in cache, get from repository
	activityLog2, err := r.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := r.cache.Set(ctx, cacheKey, activityLog2, 1*time.Hour); err != nil {
		r.logger.WithError(err).WithField("activity_log_id", id).
			Warn("Failed to cache activity log after retrieval")
	}

	return activityLog2, nil
}

func (r *CachedActivityLogRepository) GetByCompanyID(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	// Check cache for activity logs
	cacheKey := cache.BuildCompanyActivityLogsCacheKey(companyID, page, limit)
	var cachedResult struct {
		ActivityLogs []*entity.ActivityLog `json:"activity_logs"`
		Total        int                   `json:"total"`
	}

	if err := r.cache.Get(ctx, cacheKey, &cachedResult); err == nil {
		r.logger.WithFields(logrus.Fields{
			"company_id": companyID,
			"page":       page,
			"limit":      limit,
		}).Debug("Company activity logs retrieved from cache")
		return cachedResult.ActivityLogs, cachedResult.Total, nil
	}

	// If not in cache, get from repository
	activityLogs, total, err := r.repo.GetByCompanyID(ctx, companyID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Cache the result
	result := struct {
		ActivityLogs []*entity.ActivityLog `json:"activity_logs"`
		Total        int                   `json:"total"`
	}{
		ActivityLogs: activityLogs,
		Total:        total,
	}

	if err := r.cache.Set(ctx, cacheKey, result, 30*time.Minute); err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"company_id": companyID,
			"page":       page,
			"limit":      limit,
		}).Warn("Failed to cache company activity logs")
	}

	// Also cache individual activity logs
	for _, log := range activityLogs {
		individualCacheKey := cache.BuildActivityLogCacheKey(string(log.ID))
		if err := r.cache.Set(ctx, individualCacheKey, log, 1*time.Hour); err != nil {
			r.logger.WithError(err).WithField("activity_log_id", log.ID).
				Warn("Failed to cache individual activity log")
		}
	}

	return activityLogs, total, nil
}

func (r *CachedActivityLogRepository) Update(ctx context.Context, activityLog *entity.ActivityLog) error {
	// First update in the main repository
	if err := r.repo.Update(ctx, activityLog); err != nil {
		return err
	}

	// Update the cache
	cacheKey := cache.BuildActivityLogCacheKey(string(activityLog.ID))
	if err := r.cache.Set(ctx, cacheKey, activityLog, 1*time.Hour); err != nil {
		r.logger.WithError(err).WithField("activity_log_id", activityLog.ID).
			Warn("Failed to update cache after activity log update")
	}

	// Invalidate company activity logs cache
	if err := r.invalidateCompanyCache(ctx, activityLog.CompanyID); err != nil {
		r.logger.WithError(err).WithField("company_id", activityLog.CompanyID).
			Warn("Failed to invalidate company cache after update")
	}

	return nil
}

func (r *CachedActivityLogRepository) Delete(ctx context.Context, id valueobject.ActivityLogID) error {
	// First, try to get the activity log to get company ID for cache invalidation
	activityLog, err := r.GetByID(ctx, id)
	if err != nil {
		r.logger.WithError(err).WithField("activity_log_id", id).
			Warn("Failed to get activity log for cache invalidation before deletion")
	}

	// Delete from the main repository
	if err := r.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Remove from cache
	cacheKey := cache.BuildActivityLogCacheKey(string(id))
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.WithError(err).WithField("activity_log_id", id).
			Warn("Failed to delete activity log from cache")
	}

	// Invalidate company activity logs cache if we have the company ID
	if activityLog != nil {
		if err := r.invalidateCompanyCache(ctx, activityLog.CompanyID); err != nil {
			r.logger.WithError(err).WithField("company_id", activityLog.CompanyID).
				Warn("Failed to invalidate company cache after deletion")
		}
	}

	return nil
}

func (r *CachedActivityLogRepository) GetByObjectID(ctx context.Context, companyID, objectID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	// For now, we'll not cache this method to keep it simple
	// In a production system, you might want to cache this as well
	return r.repo.GetByObjectID(ctx, companyID, objectID, page, limit)
}

func (r *CachedActivityLogRepository) GetByActivityName(ctx context.Context, companyID, activityName string, page, limit int) ([]*entity.ActivityLog, int, error) {
	// For now, we'll not cache this method to keep it simple
	// In a production system, you might want to cache this as well
	return r.repo.GetByActivityName(ctx, companyID, activityName, page, limit)
}

func (r *CachedActivityLogRepository) GetByDateRange(ctx context.Context, companyID string, startDate, endDate time.Time, page, limit int) ([]*entity.ActivityLog, int, error) {
	// For now, we'll not cache this method to keep it simple
	// In a production system, you might want to cache this as well
	return r.repo.GetByDateRange(ctx, companyID, startDate, endDate, page, limit)
}

func (r *CachedActivityLogRepository) GetByActor(ctx context.Context, companyID, actorID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	// For now, we'll not cache this method to keep it simple
	// In a production system, you might want to cache this as well
	return r.repo.GetByActor(ctx, companyID, actorID, page, limit)
}

func (r *CachedActivityLogRepository) CountByCompanyID(ctx context.Context, companyID string) (int, error) {
	// Check cache for count
	cacheKey := cache.BuildActivityLogCountCacheKey(companyID)
	var count int
	if err := r.cache.Get(ctx, cacheKey, &count); err == nil {
		r.logger.WithField("company_id", companyID).Debug("Activity log count retrieved from cache")
		return count, nil
	}

	// If not in cache, get from repository
	count, err := r.repo.CountByCompanyID(ctx, companyID)
	if err != nil {
		return 0, err
	}

	// Cache the result for 5 minutes
	if err := r.cache.Set(ctx, cacheKey, count, 5*time.Minute); err != nil {
		r.logger.WithError(err).WithField("company_id", companyID).
			Warn("Failed to cache activity log count")
	}

	return count, nil
}

// invalidateCompanyCache invalidates all cached data for a company
func (r *CachedActivityLogRepository) invalidateCompanyCache(ctx context.Context, companyID string) error {
	// Delete company activity logs cache patterns
	pattern := fmt.Sprintf("company_activity_logs:%s:*", companyID)
	if err := r.cache.DeleteByPattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to delete company activity logs cache: %w", err)
	}

	// Delete company count cache
	countKey := cache.BuildActivityLogCountCacheKey(companyID)
	if err := r.cache.Delete(ctx, countKey); err != nil {
		return fmt.Errorf("failed to delete company count cache: %w", err)
	}

	return nil
}

// ClearCache clears all cached data
func (r *CachedActivityLogRepository) ClearCache(ctx context.Context) error {
	return r.cache.FlushAll(ctx)
}

// ClearCacheForCompany clears all cached data for a specific company
func (r *CachedActivityLogRepository) ClearCacheForCompany(ctx context.Context, companyID string) error {
	return r.invalidateCompanyCache(ctx, companyID)
}
