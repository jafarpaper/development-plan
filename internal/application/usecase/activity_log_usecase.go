package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/event"
	"activity-log-service/internal/domain/repository"
	"activity-log-service/internal/domain/valueobject"
	"activity-log-service/internal/infrastructure/email"
	"activity-log-service/internal/infrastructure/messaging"
)

type ActivityLogUseCase struct {
	arangoRepo repository.ActivityLogRepository
	publisher  *messaging.NATSPublisher
	mailer     *email.Mailer
}

func NewActivityLogUseCase(
	arangoRepo repository.ActivityLogRepository,
	publisher *messaging.NATSPublisher,
	mailer *email.Mailer,
) *ActivityLogUseCase {
	return &ActivityLogUseCase{
		arangoRepo: arangoRepo,
		publisher:  publisher,
		mailer:     mailer,
	}
}

func (uc *ActivityLogUseCase) CreateActivityLog(ctx context.Context, req *CreateActivityLogRequest) (*entity.ActivityLog, error) {
	var changes json.RawMessage
	if req.Changes != "" {
		if !json.Valid([]byte(req.Changes)) {
			return nil, fmt.Errorf("invalid JSON in changes field")
		}
		changes = json.RawMessage(req.Changes)
	}

	activityLog := entity.NewActivityLog(
		req.ActivityName,
		req.CompanyID,
		req.ObjectName,
		req.ObjectID,
		changes,
		req.FormattedMessage,
		req.ActorID,
		req.ActorName,
		req.ActorEmail,
	)

	if err := activityLog.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid activity log: %w", err)
	}

	if err := uc.arangoRepo.Create(ctx, activityLog); err != nil {
		return nil, fmt.Errorf("failed to create activity log: %w", err)
	}

	if uc.publisher != nil {
		event := event.NewActivityLogCreated(activityLog)
		if err := uc.publisher.PublishActivityLogCreated(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish event: %w", err)
		}
	}

	// Send email notification if configured
	if uc.mailer != nil {
		go func() {
			emailData := email.ActivityLogEmailData{
				ActivityLog: activityLog,
				CompanyName: fmt.Sprintf("Company %s", activityLog.CompanyID),
				Recipients:  []string{activityLog.ActorEmail},
				Subject:     fmt.Sprintf("Activity Log: %s", activityLog.FormattedMessage),
			}
			if err := uc.mailer.SendActivityLogNotification(context.Background(), emailData); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Failed to send email notification: %v\n", err)
			}
		}()
	}

	return activityLog, nil
}

func (uc *ActivityLogUseCase) GetActivityLog(ctx context.Context, id string) (*entity.ActivityLog, error) {
	activityLogID := valueobject.ActivityLogID(id)
	if !activityLogID.IsValid() {
		return nil, fmt.Errorf("invalid activity log ID")
	}

	activityLog, err := uc.arangoRepo.GetByID(ctx, activityLogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity log: %w", err)
	}

	return activityLog, nil
}

func (uc *ActivityLogUseCase) ListActivityLogs(ctx context.Context, companyID string, page, limit int) ([]*entity.ActivityLog, int, error) {
	if companyID == "" {
		return nil, 0, fmt.Errorf("company ID is required")
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	activityLogs, total, err := uc.arangoRepo.GetByCompanyID(ctx, companyID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list activity logs: %w", err)
	}

	return activityLogs, total, nil
}

type CreateActivityLogRequest struct {
	ActivityName     string `json:"activity_name"`
	CompanyID        string `json:"company_id"`
	ObjectName       string `json:"object_name"`
	ObjectID         string `json:"object_id"`
	Changes          string `json:"changes"`
	FormattedMessage string `json:"formatted_message"`
	ActorID          string `json:"actor_id"`
	ActorName        string `json:"actor_name"`
	ActorEmail       string `json:"actor_email"`
}
