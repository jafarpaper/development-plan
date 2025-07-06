package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"activity-log-service/internal/application/usecase"
	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/valueobject"
)

// InMemoryRepository for testing
type InMemoryRepository struct {
	logs map[string]*entity.ActivityLog
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		logs: make(map[string]*entity.ActivityLog),
	}
}

func (r *InMemoryRepository) Create(ctx context.Context, activityLog *entity.ActivityLog) error {
	r.logs[string(activityLog.ID)] = activityLog
	return nil
}

func (r *InMemoryRepository) GetByID(ctx context.Context, id valueobject.ActivityLogID) (*entity.ActivityLog, error) {
	log, exists := r.logs[string(id)]
	if !exists {
		return nil, entity.ErrActivityLogNotFound
	}
	return log, nil
}

func (r *InMemoryRepository) GetByCompanyID(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	var result []*entity.ActivityLog
	for _, log := range r.logs {
		if log.CompanyID == companyID {
			result = append(result, log)
		}
	}
	return result, len(result), nil
}

func (r *InMemoryRepository) Update(ctx context.Context, activityLog *entity.ActivityLog) error {
	r.logs[string(activityLog.ID)] = activityLog
	return nil
}

func (r *InMemoryRepository) Delete(ctx context.Context, id valueobject.ActivityLogID) error {
	delete(r.logs, string(id))
	return nil
}

func (r *InMemoryRepository) GetByObjectID(ctx context.Context, companyID, objectID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	return nil, 0, nil
}

func (r *InMemoryRepository) GetByActivityName(ctx context.Context, companyID, activityName string, page, limit int) ([]*entity.ActivityLog, int, error) {
	return nil, 0, nil
}

func (r *InMemoryRepository) GetByDateRange(ctx context.Context, companyID string, startDate, endDate time.Time, page, limit int) ([]*entity.ActivityLog, int, error) {
	return nil, 0, nil
}

func (r *InMemoryRepository) GetByActor(ctx context.Context, companyID, actorID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	return nil, 0, nil
}

func (r *InMemoryRepository) CountByCompanyID(ctx context.Context, companyID string) (int, error) {
	count := 0
	for _, log := range r.logs {
		if log.CompanyID == companyID {
			count++
		}
	}
	return count, nil
}

// NoOpPublisher for testing
type NoOpPublisher struct{}

func (p *NoOpPublisher) PublishActivityLogCreated(ctx context.Context, event interface{}) error {
	return nil
}

func (p *NoOpPublisher) Close() error {
	return nil
}

func (p *NoOpPublisher) EnsureStream(streamName, subject string) error {
	return nil
}

func main() {
	// Test basic functionality
	repo := NewInMemoryRepository()
	uc := usecase.NewActivityLogUseCase(repo, nil, nil)

	ctx := context.Background()

	// Test 1: Create Activity Log
	fmt.Println("Test 1: Creating activity log...")
	req := &usecase.CreateActivityLogRequest{
		ActivityName:     "user_created",
		CompanyID:        "company1",
		ObjectName:       "user",
		ObjectID:         "user123",
		Changes:          `{"name": "John Doe", "email": "john@example.com"}`,
		FormattedMessage: "User John Doe was created",
		ActorID:          "actor1",
		ActorName:        "Admin User",
		ActorEmail:       "admin@example.com",
	}

	activityLog, err := uc.CreateActivityLog(ctx, req)
	if err != nil {
		fmt.Printf("❌ Create test failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Create test passed - ID: %s\n", activityLog.ID)

	// Test 2: Get Activity Log
	fmt.Println("Test 2: Getting activity log...")
	retrievedLog, err := uc.GetActivityLog(ctx, string(activityLog.ID))
	if err != nil {
		fmt.Printf("❌ Get test failed: %v\n", err)
		os.Exit(1)
	}
	if retrievedLog.ID != activityLog.ID {
		fmt.Printf("❌ Get test failed: IDs don't match\n")
		os.Exit(1)
	}
	fmt.Printf("✅ Get test passed - Retrieved ID: %s\n", retrievedLog.ID)

	// Test 3: List Activity Logs
	fmt.Println("Test 3: Listing activity logs...")
	_, total, err := uc.ListActivityLogs(ctx, "company1", 1, 10)
	if err != nil {
		fmt.Printf("❌ List test failed: %v\n", err)
		os.Exit(1)
	}
	if total != 1 {
		fmt.Printf("❌ List test failed: Expected 1 log, got %d\n", total)
		os.Exit(1)
	}
	fmt.Printf("✅ List test passed - Found %d logs\n", total)

	// Test 4: Test invalid actor email
	fmt.Println("Test 4: Testing invalid actor email...")
	invalidReq := &usecase.CreateActivityLogRequest{
		ActivityName:     "user_created",
		CompanyID:        "company1",
		ObjectName:       "user",
		ObjectID:         "user123",
		Changes:          `{"name": "Jane Doe"}`,
		FormattedMessage: "User Jane Doe was created",
		ActorID:          "actor2",
		ActorName:        "Admin User",
		ActorEmail:       "invalid-email",
	}

	_, err = uc.CreateActivityLog(ctx, invalidReq)
	if err == nil {
		fmt.Printf("❌ Invalid email test failed: Expected error but got none\n")
		os.Exit(1)
	}
	fmt.Printf("✅ Invalid email test passed - Got expected error: %v\n", err)

	// Test 5: Test invalid JSON
	fmt.Println("Test 5: Testing invalid JSON...")
	invalidJSONReq := &usecase.CreateActivityLogRequest{
		ActivityName:     "user_created",
		CompanyID:        "company1",
		ObjectName:       "user",
		ObjectID:         "user123",
		Changes:          `{"name": "Jane Doe"`,
		FormattedMessage: "User Jane Doe was created",
		ActorID:          "actor2",
		ActorName:        "Admin User",
		ActorEmail:       "admin@example.com",
	}

	_, err = uc.CreateActivityLog(ctx, invalidJSONReq)
	if err == nil {
		fmt.Printf("❌ Invalid JSON test failed: Expected error but got none\n")
		os.Exit(1)
	}
	fmt.Printf("✅ Invalid JSON test passed - Got expected error: %v\n", err)

	// Test 6: Test entity validation
	fmt.Println("Test 6: Testing entity validation...")
	actor, err := valueobject.NewActor("test-id", "Test User", "test@example.com")
	if err != nil {
		fmt.Printf("❌ Actor creation failed: %v\n", err)
		os.Exit(1)
	}

	testLog := entity.NewActivityLog(
		"test_activity",
		"test-company",
		"test-object",
		"test-object-id",
		json.RawMessage(`{"test": "data"}`),
		"Test activity occurred",
		actor.ID,
		actor.Name,
		actor.Email,
	)

	if err := testLog.IsValid(); err != nil {
		fmt.Printf("❌ Entity validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Entity validation passed\n")

	// Test 7: Test invalid entity
	fmt.Println("Test 7: Testing invalid entity...")
	invalidLog := &entity.ActivityLog{
		ID:               valueobject.NewActivityLogID(),
		ActivityName:     "", // Invalid - empty
		CompanyID:        "test-company",
		ObjectName:       "test-object",
		ObjectID:         "test-object-id",
		FormattedMessage: "Test message",
		ActorID:          actor.ID,
		ActorName:        actor.Name,
		ActorEmail:       actor.Email,
	}

	if err := invalidLog.IsValid(); err == nil {
		fmt.Printf("❌ Invalid entity test failed: Expected error but got none\n")
		os.Exit(1)
	}
	fmt.Printf("✅ Invalid entity test passed - Got expected error\n")

	fmt.Println("\n🎉 All tests passed! Unit tests are working correctly.")
}
