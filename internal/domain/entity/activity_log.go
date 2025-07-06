package entity

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"activity-log-service/internal/domain/valueobject"
)

var (
	ErrInvalidActorID    = errors.New("invalid actor id")
	ErrInvalidActorName  = errors.New("invalid actor name")
	ErrInvalidActorEmail = errors.New("invalid actor email")
)

type ActivityLog struct {
	ID               valueobject.ActivityLogID `json:"id" arango:"_key"`
	ActivityName     string                    `json:"activity_name"`
	CompanyID        string                    `json:"company_id"`
	ObjectName       string                    `json:"object_name"`
	ObjectID         string                    `json:"object_id"`
	Changes          json.RawMessage           `json:"changes"`
	FormattedMessage string                    `json:"formatted_message"`
	ActorID          string                    `json:"actor_id"`
	ActorName        string                    `json:"actor_name"`
	ActorEmail       string                    `json:"actor_email"`
	CreatedAt        time.Time                 `json:"created_at"`
}

func NewActivityLog(
	activityName string,
	companyID string,
	objectName string,
	objectID string,
	changes json.RawMessage,
	formattedMessage string,
	actorID string,
	actorName string,
	actorEmail string,
) *ActivityLog {
	return &ActivityLog{
		ID:               valueobject.NewActivityLogID(),
		ActivityName:     activityName,
		CompanyID:        companyID,
		ObjectName:       objectName,
		ObjectID:         objectID,
		Changes:          changes,
		FormattedMessage: formattedMessage,
		ActorID:          actorID,
		ActorName:        actorName,
		ActorEmail:       actorEmail,
		CreatedAt:        time.Now().UTC(),
	}
}

func (al *ActivityLog) IsValid() error {
	if al.ActivityName == "" {
		return ErrInvalidActivityName
	}
	if al.CompanyID == "" {
		return ErrInvalidCompanyID
	}
	if al.ObjectName == "" {
		return ErrInvalidObjectName
	}
	if al.ObjectID == "" {
		return ErrInvalidObjectID
	}
	if al.FormattedMessage == "" {
		return ErrInvalidFormattedMessage
	}
	if al.ActorID == "" {
		return ErrInvalidActorID
	}
	if al.ActorName == "" {
		return ErrInvalidActorName
	}
	if !isValidEmail(al.ActorEmail) {
		return ErrInvalidActorEmail
	}
	return nil
}

func (al *ActivityLog) ToJSON() ([]byte, error) {
	return json.Marshal(al)
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
