package activitylog

import (
    "context"
    "fmt"

    driver "github.com/arangodb/go-driver"
)

type Repository interface {
    Create(ctx context.Context, log *ActivityLog) error
    GetByID(ctx context.Context, id string) (*ActivityLog, error)
}

type repository struct {
    col driver.Collection
}

func NewRepository(db driver.Database) Repository {
    col, _ := db.Collection(context.Background(), "activity_log")
    return &repository{col: col}
}

func (r *repository) Create(ctx context.Context, log *ActivityLog) error {
    meta, err := r.col.CreateDocument(ctx, log)
    if err != nil {
        return fmt.Errorf("failed to create activity log: %w", err)
    }
    log.ID = meta.Key
    return nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*ActivityLog, error) {
    var log ActivityLog
    _, err := r.col.ReadDocument(ctx, id, &log)
    if err != nil {
        return nil, fmt.Errorf("failed to get activity log: %w", err)
    }
    return &log, nil
}
