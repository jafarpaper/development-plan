package entity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"activity-log-service/internal/domain/valueobject"
)

func TestNewActivityLog(t *testing.T) {
	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	changes := json.RawMessage(`{"field": "value"}`)

	activityLog := NewActivityLog(
		"user_created",
		"company1",
		"user",
		"user123",
		changes,
		"User John Doe was created",
		actor.ID,
		actor.Name,
		actor.Email,
	)

	assert.NotEmpty(t, activityLog.ID)
	assert.Equal(t, "user_created", activityLog.ActivityName)
	assert.Equal(t, "company1", activityLog.CompanyID)
	assert.Equal(t, "user", activityLog.ObjectName)
	assert.Equal(t, "user123", activityLog.ObjectID)
	assert.Equal(t, changes, activityLog.Changes)
	assert.Equal(t, "User John Doe was created", activityLog.FormattedMessage)
	assert.Equal(t, actor.ID, activityLog.ActorID)
	assert.Equal(t, actor.Name, activityLog.ActorName)
	assert.Equal(t, actor.Email, activityLog.ActorEmail)
	assert.WithinDuration(t, time.Now(), activityLog.CreatedAt, time.Second)
}

func TestActivityLog_IsValid(t *testing.T) {
	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	tests := []struct {
		name        string
		activityLog *ActivityLog
		wantErr     error
	}{
		{
			name: "valid activity log",
			activityLog: &ActivityLog{
				ID:               valueobject.NewActivityLogID(),
				ActivityName:     "user_created",
				CompanyID:        "company1",
				ObjectName:       "user",
				ObjectID:         "user123",
				Changes:          json.RawMessage(`{"field": "value"}`),
				FormattedMessage: "User created",
				ActorID:          actor.ID,
				ActorName:        actor.Name,
				ActorEmail:       actor.Email,
				CreatedAt:        time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "empty activity name",
			activityLog: &ActivityLog{
				ID:               valueobject.NewActivityLogID(),
				ActivityName:     "",
				CompanyID:        "company1",
				ObjectName:       "user",
				ObjectID:         "user123",
				Changes:          json.RawMessage(`{"field": "value"}`),
				FormattedMessage: "User created",
				ActorID:          actor.ID,
				ActorName:        actor.Name,
				ActorEmail:       actor.Email,
				CreatedAt:        time.Now(),
			},
			wantErr: ErrInvalidActivityName,
		},
		{
			name: "empty company id",
			activityLog: &ActivityLog{
				ID:               valueobject.NewActivityLogID(),
				ActivityName:     "user_created",
				CompanyID:        "",
				ObjectName:       "user",
				ObjectID:         "user123",
				Changes:          json.RawMessage(`{"field": "value"}`),
				FormattedMessage: "User created",
				ActorID:          actor.ID,
				ActorName:        actor.Name,
				ActorEmail:       actor.Email,
				CreatedAt:        time.Now(),
			},
			wantErr: ErrInvalidCompanyID,
		},
		{
			name: "empty object name",
			activityLog: &ActivityLog{
				ID:               valueobject.NewActivityLogID(),
				ActivityName:     "user_created",
				CompanyID:        "company1",
				ObjectName:       "",
				ObjectID:         "user123",
				Changes:          json.RawMessage(`{"field": "value"}`),
				FormattedMessage: "User created",
				ActorID:          actor.ID,
				ActorName:        actor.Name,
				ActorEmail:       actor.Email,
				CreatedAt:        time.Now(),
			},
			wantErr: ErrInvalidObjectName,
		},
		{
			name: "empty object id",
			activityLog: &ActivityLog{
				ID:               valueobject.NewActivityLogID(),
				ActivityName:     "user_created",
				CompanyID:        "company1",
				ObjectName:       "user",
				ObjectID:         "",
				Changes:          json.RawMessage(`{"field": "value"}`),
				FormattedMessage: "User created",
				ActorID:          actor.ID,
				ActorName:        actor.Name,
				ActorEmail:       actor.Email,
				CreatedAt:        time.Now(),
			},
			wantErr: ErrInvalidObjectID,
		},
		{
			name: "empty formatted message",
			activityLog: &ActivityLog{
				ID:               valueobject.NewActivityLogID(),
				ActivityName:     "user_created",
				CompanyID:        "company1",
				ObjectName:       "user",
				ObjectID:         "user123",
				Changes:          json.RawMessage(`{"field": "value"}`),
				FormattedMessage: "",
				ActorID:          actor.ID,
				ActorName:        actor.Name,
				ActorEmail:       actor.Email,
				CreatedAt:        time.Now(),
			},
			wantErr: ErrInvalidFormattedMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.activityLog.IsValid()
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestActivityLog_ToJSON(t *testing.T) {
	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	require.NoError(t, err)

	activityLog := NewActivityLog(
		"user_created",
		"company1",
		"user",
		"user123",
		json.RawMessage(`{"field": "value"}`),
		"User John Doe was created",
		actor.ID,
		actor.Name,
		actor.Email,
	)

	jsonData, err := activityLog.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var parsedLog ActivityLog
	err = json.Unmarshal(jsonData, &parsedLog)
	require.NoError(t, err)

	assert.Equal(t, activityLog.ID, parsedLog.ID)
	assert.Equal(t, activityLog.ActivityName, parsedLog.ActivityName)
	assert.Equal(t, activityLog.CompanyID, parsedLog.CompanyID)
}
