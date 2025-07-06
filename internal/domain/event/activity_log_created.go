package event

import (
	"encoding/json"
	"time"

	"activity-log-service/internal/domain/entity"
)

type ActivityLogCreated struct {
	EventID     string              `json:"event_id"`
	EventType   string              `json:"event_type"`
	AggregateID string              `json:"aggregate_id"`
	ActivityLog *entity.ActivityLog `json:"activity_log"`
	Timestamp   time.Time           `json:"timestamp"`
	Version     int                 `json:"version"`
}

func NewActivityLogCreated(activityLog *entity.ActivityLog) *ActivityLogCreated {
	return &ActivityLogCreated{
		EventID:     generateEventID(),
		EventType:   "activity_log_created",
		AggregateID: activityLog.ID.String(),
		ActivityLog: activityLog,
		Timestamp:   time.Now().UTC(),
		Version:     1,
	}
}

func (e *ActivityLogCreated) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e *ActivityLogCreated) GetEventType() string {
	return e.EventType
}

func (e *ActivityLogCreated) GetAggregateID() string {
	return e.AggregateID
}

func (e *ActivityLogCreated) GetTimestamp() time.Time {
	return e.Timestamp
}

func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + randString(8)
}

func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
