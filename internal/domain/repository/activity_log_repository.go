package repository

import (
	"context"
	"time"

	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/valueobject"
)

type ActivityLogRepository interface {
	Create(ctx context.Context, activityLog *entity.ActivityLog) error
	GetByID(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error)
	GetByCompanyID(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error)
	Update(ctx context.Context, activityLog *entity.ActivityLog) error
	Delete(ctx context.Context, id valueobject.ActivityLogID) error
	GetByObjectID(ctx context.Context, companyID, objectID string, page, limit int) ([]*entity.ActivityLog, int, error)
	GetByActivityName(ctx context.Context, companyID, activityName string, page, limit int) ([]*entity.ActivityLog, int, error)
	GetByDateRange(ctx context.Context, companyID string, startDate, endDate time.Time, page, limit int) ([]*entity.ActivityLog, int, error)
	GetByActor(ctx context.Context, companyID, actorID string, page, limit int) ([]*entity.ActivityLog, int, error)
	CountByCompanyID(ctx context.Context, companyID string) (int, error)
}
