package database

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"

	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/repository"
	"activity-log-service/internal/domain/valueobject"
)

type ArangoActivityLogRepository struct {
	client     driver.Client
	database   driver.Database
	collection driver.Collection
}

func NewArangoActivityLogRepository(url, dbName, collectionName, username, password string) (*ArangoActivityLogRepository, error) {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{url},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(username, password),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	ctx := context.Background()

	db, err := client.Database(ctx, dbName)
	if driver.IsNotFound(err) {
		db, err = client.CreateDatabase(ctx, dbName, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	collection, err := db.Collection(ctx, collectionName)
	if driver.IsNotFound(err) {
		collection, err = db.CreateCollection(ctx, collectionName, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to open collection: %w", err)
	}

	return &ArangoActivityLogRepository{
		client:     client,
		database:   db,
		collection: collection,
	}, nil
}

func (r *ArangoActivityLogRepository) Create(ctx context.Context, activityLog *entity.ActivityLog) error {
	_, err := r.collection.CreateDocument(ctx, activityLog)
	if err != nil {
		return fmt.Errorf("failed to create activity log: %w", err)
	}
	return nil
}

func (r *ArangoActivityLogRepository) GetByID(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error) {
	var activityLog entity.ActivityLog
	_, err := r.collection.ReadDocument(ctx, id.String(), &activityLog)
	if driver.IsNotFound(err) {
		return nil, entity.ErrActivityLogNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read activity log: %w", err)
	}
	return &activityLog, nil
}

func (r *ArangoActivityLogRepository) GetByCompanyID(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	offset := (page - 1) * limit

	query := `
		FOR log IN @@collection
		FILTER log.company_id == @companyId
		SORT log.created_at DESC
		LIMIT @offset, @limit
		RETURN log
	`

	bindVars := map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyId":   companyID,
		"offset":      offset,
		"limit":       limit,
	}

	cursor, err := r.database.Query(ctx, query, bindVars)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query activity logs: %w", err)
	}
	defer cursor.Close()

	var logs []*entity.ActivityLog
	for cursor.HasMore() {
		var log entity.ActivityLog
		if _, err := cursor.ReadDocument(ctx, &log); err != nil {
			return nil, 0, fmt.Errorf("failed to read document: %w", err)
		}
		logs = append(logs, &log)
	}

	countQuery := `
		FOR log IN @@collection
		FILTER log.company_id == @companyId
		COLLECT WITH COUNT INTO total
		RETURN total
	`

	countBindVars := map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyId":   companyID,
	}

	countCursor, err := r.database.Query(ctx, countQuery, countBindVars)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count activity logs: %w", err)
	}
	defer countCursor.Close()

	var total int
	if countCursor.HasMore() {
		if _, err := countCursor.ReadDocument(ctx, &total); err != nil {
			return nil, 0, fmt.Errorf("failed to read count: %w", err)
		}
	}

	return logs, total, nil
}

func (r *ArangoActivityLogRepository) Update(ctx context.Context, activityLog *entity.ActivityLog) error {
	_, err := r.collection.UpdateDocument(ctx, activityLog.ID.String(), activityLog)
	if driver.IsNotFound(err) {
		return entity.ErrActivityLogNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update activity log: %w", err)
	}
	return nil
}

func (r *ArangoActivityLogRepository) Delete(ctx context.Context, id valueobject.ActivityLogID) error {
	_, err := r.collection.RemoveDocument(ctx, id.String())
	if driver.IsNotFound(err) {
		return entity.ErrActivityLogNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to delete activity log: %w", err)
	}
	return nil
}

func (r *ArangoActivityLogRepository) GetByObjectID(ctx context.Context, companyID, objectID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	offset := (page - 1) * limit
	query := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID AND log.object_id == @objectID
		SORT log.created_at DESC
		LIMIT @offset, @limit
		RETURN log
	`
	bindVars := map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyID":   companyID,
		"objectID":    objectID,
		"offset":      offset,
		"limit":       limit,
	}

	cursor, err := r.database.Query(ctx, query, bindVars)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query activity logs by object ID: %w", err)
	}
	defer cursor.Close()

	var logs []*entity.ActivityLog
	for cursor.HasMore() {
		var log entity.ActivityLog
		_, err := cursor.ReadDocument(ctx, &log)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read document: %w", err)
		}
		logs = append(logs, &log)
	}

	// Get total count
	countQuery := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID AND log.object_id == @objectID
		COLLECT WITH COUNT INTO total
		RETURN total
	`
	countCursor, err := r.database.Query(ctx, countQuery, map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyID":   companyID,
		"objectID":    objectID,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count activity logs: %w", err)
	}
	defer countCursor.Close()

	var total int
	if countCursor.HasMore() {
		_, err := countCursor.ReadDocument(ctx, &total)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read count: %w", err)
		}
	}

	return logs, total, nil
}

func (r *ArangoActivityLogRepository) GetByActivityName(ctx context.Context, companyID, activityName string, page, limit int) ([]*entity.ActivityLog, int, error) {
	offset := (page - 1) * limit
	query := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID AND log.activity_name == @activityName
		SORT log.created_at DESC
		LIMIT @offset, @limit
		RETURN log
	`
	bindVars := map[string]interface{}{
		"@collection":  r.collection.Name(),
		"companyID":    companyID,
		"activityName": activityName,
		"offset":       offset,
		"limit":        limit,
	}

	cursor, err := r.database.Query(ctx, query, bindVars)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query activity logs by activity name: %w", err)
	}
	defer cursor.Close()

	var logs []*entity.ActivityLog
	for cursor.HasMore() {
		var log entity.ActivityLog
		_, err := cursor.ReadDocument(ctx, &log)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read document: %w", err)
		}
		logs = append(logs, &log)
	}

	// Get total count
	countQuery := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID AND log.activity_name == @activityName
		COLLECT WITH COUNT INTO total
		RETURN total
	`
	countCursor, err := r.database.Query(ctx, countQuery, map[string]interface{}{
		"@collection":  r.collection.Name(),
		"companyID":    companyID,
		"activityName": activityName,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count activity logs: %w", err)
	}
	defer countCursor.Close()

	var total int
	if countCursor.HasMore() {
		_, err := countCursor.ReadDocument(ctx, &total)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read count: %w", err)
		}
	}

	return logs, total, nil
}

func (r *ArangoActivityLogRepository) GetByDateRange(ctx context.Context, companyID string, startDate, endDate time.Time, page, limit int) ([]*entity.ActivityLog, int, error) {
	offset := (page - 1) * limit
	query := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID AND log.created_at >= @startDate AND log.created_at <= @endDate
		SORT log.created_at DESC
		LIMIT @offset, @limit
		RETURN log
	`
	bindVars := map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyID":   companyID,
		"startDate":   startDate,
		"endDate":     endDate,
		"offset":      offset,
		"limit":       limit,
	}

	cursor, err := r.database.Query(ctx, query, bindVars)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query activity logs by date range: %w", err)
	}
	defer cursor.Close()

	var logs []*entity.ActivityLog
	for cursor.HasMore() {
		var log entity.ActivityLog
		_, err := cursor.ReadDocument(ctx, &log)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read document: %w", err)
		}
		logs = append(logs, &log)
	}

	// Get total count
	countQuery := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID AND log.created_at >= @startDate AND log.created_at <= @endDate
		COLLECT WITH COUNT INTO total
		RETURN total
	`
	countCursor, err := r.database.Query(ctx, countQuery, map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyID":   companyID,
		"startDate":   startDate,
		"endDate":     endDate,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count activity logs: %w", err)
	}
	defer countCursor.Close()

	var total int
	if countCursor.HasMore() {
		_, err := countCursor.ReadDocument(ctx, &total)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read count: %w", err)
		}
	}

	return logs, total, nil
}

func (r *ArangoActivityLogRepository) GetByActor(ctx context.Context, companyID, actorID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	offset := (page - 1) * limit
	query := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID AND log.actor_id == @actorID
		SORT log.created_at DESC
		LIMIT @offset, @limit
		RETURN log
	`
	bindVars := map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyID":   companyID,
		"actorID":     actorID,
		"offset":      offset,
		"limit":       limit,
	}

	cursor, err := r.database.Query(ctx, query, bindVars)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query activity logs by actor: %w", err)
	}
	defer cursor.Close()

	var logs []*entity.ActivityLog
	for cursor.HasMore() {
		var log entity.ActivityLog
		_, err := cursor.ReadDocument(ctx, &log)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read document: %w", err)
		}
		logs = append(logs, &log)
	}

	// Get total count
	countQuery := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID AND log.actor_id == @actorID
		COLLECT WITH COUNT INTO total
		RETURN total
	`
	countCursor, err := r.database.Query(ctx, countQuery, map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyID":   companyID,
		"actorID":     actorID,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count activity logs: %w", err)
	}
	defer countCursor.Close()

	var total int
	if countCursor.HasMore() {
		_, err := countCursor.ReadDocument(ctx, &total)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read count: %w", err)
		}
	}

	return logs, total, nil
}

func (r *ArangoActivityLogRepository) CountByCompanyID(ctx context.Context, companyID string) (int, error) {
	query := `
		FOR log IN @@collection
		FILTER log.company_id == @companyID
		COLLECT WITH COUNT INTO total
		RETURN total
	`
	bindVars := map[string]interface{}{
		"@collection": r.collection.Name(),
		"companyID":   companyID,
	}

	cursor, err := r.database.Query(ctx, query, bindVars)
	if err != nil {
		return 0, fmt.Errorf("failed to count activity logs by company ID: %w", err)
	}
	defer cursor.Close()

	var total int
	if cursor.HasMore() {
		_, err := cursor.ReadDocument(ctx, &total)
		if err != nil {
			return 0, fmt.Errorf("failed to read count: %w", err)
		}
	}

	return total, nil
}

var _ repository.ActivityLogRepository = (*ArangoActivityLogRepository)(nil)
