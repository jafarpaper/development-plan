package email

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"activity-log-service/internal/domain/entity"
	"activity-log-service/internal/domain/valueobject"
)

func TestNewMailer(t *testing.T) {
	logger := logrus.New()
	config := EmailConfig{
		Host:     "localhost",
		Port:     1025,
		Username: "",
		Password: "",
		From:     "test@example.com",
	}

	mailer := NewMailer(config, logger)
	assert.NotNil(t, mailer)
	assert.NotNil(t, mailer.dialer)
	assert.Equal(t, config.From, mailer.from)
	assert.NotNil(t, mailer.logger)
	assert.NotNil(t, mailer.templates)
}

func TestMailer_LoadTemplates(t *testing.T) {
	logger := logrus.New()
	config := EmailConfig{
		Host: "localhost",
		Port: 1025,
		From: "test@example.com",
	}

	mailer := NewMailer(config, logger)

	// Check that templates are loaded
	assert.Contains(t, mailer.templates, "activity_log")
	assert.Contains(t, mailer.templates, "daily_summary")
}

func TestMailer_SendActivityLogNotification_NoRecipients(t *testing.T) {
	logger := logrus.New()
	config := EmailConfig{
		Host: "localhost",
		Port: 1025,
		From: "test@example.com",
	}

	mailer := NewMailer(config, logger)
	ctx := context.Background()

	actor, err := valueobject.NewActor("actor1", "John Doe", "john@example.com")
	assert.NoError(t, err)

	activityLog := entity.NewActivityLog(
		"user_created",
		"company1",
		"user",
		"user123",
		nil,
		"User was created",
		actor,
	)

	data := ActivityLogEmailData{
		ActivityLog: activityLog,
		CompanyName: "Test Company",
		Recipients:  []string{}, // Empty recipients
		Subject:     "Test Subject",
	}

	err = mailer.SendActivityLogNotification(ctx, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no recipients specified")
}

func TestMailer_SendDailySummary_NoRecipients(t *testing.T) {
	logger := logrus.New()
	config := EmailConfig{
		Host: "localhost",
		Port: 1025,
		From: "test@example.com",
	}

	mailer := NewMailer(config, logger)
	ctx := context.Background()

	summaryData := map[string]interface{}{
		"Date":            "2023-01-01",
		"TotalActivities": 10,
		"UniqueUsers":     5,
		"TopActivity":     "user_login",
	}

	err := mailer.SendDailySummary(ctx, []string{}, summaryData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no recipients specified")
}

func TestMailer_SendDailySummary_ValidData(t *testing.T) {
	// Skip this test if MailHog is not available
	t.Skip("Skipping MailHog integration test - requires running MailHog server")

	logger := logrus.New()
	config := EmailConfig{
		Host: "localhost",
		Port: 1025,
		From: "test@example.com",
	}

	mailer := NewMailer(config, logger)
	ctx := context.Background()

	summaryData := map[string]interface{}{
		"Date":            "2023-01-01",
		"TotalActivities": 10,
		"UniqueUsers":     5,
		"TopActivity":     "user_login",
	}

	recipients := []string{"admin@example.com"}

	err := mailer.SendDailySummary(ctx, recipients, summaryData)
	assert.NoError(t, err)
}
