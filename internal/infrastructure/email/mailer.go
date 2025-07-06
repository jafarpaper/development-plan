package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"

	"activity-log-service/internal/domain/entity"
)

type Mailer struct {
	dialer    *gomail.Dialer
	from      string
	logger    *logrus.Logger
	templates map[string]*template.Template
}

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type ActivityLogEmailData struct {
	ActivityLog    *entity.ActivityLog
	CompanyName    string
	Recipients     []string
	Subject        string
	WebURL         string
	UnsubscribeURL string
}

func NewMailer(config EmailConfig, logger *logrus.Logger) *Mailer {
	dialer := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)

	// For MailHog, we don't need authentication
	if config.Host == "localhost" || config.Host == "mailhog" {
		dialer.Auth = nil
	}

	mailer := &Mailer{
		dialer:    dialer,
		from:      config.From,
		logger:    logger,
		templates: make(map[string]*template.Template),
	}

	// Load email templates
	mailer.loadTemplates()

	return mailer
}

func (m *Mailer) loadTemplates() {
	// Activity log notification template
	activityLogTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Activity Log Notification</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; padding: 20px; border-radius: 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
        .header { background-color: #007bff; color: white; padding: 15px; text-align: center; border-radius: 5px 5px 0 0; margin: -20px -20px 20px -20px; }
        .activity-details { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 15px 0; }
        .detail-row { margin: 8px 0; }
        .label { font-weight: bold; color: #495057; }
        .value { color: #212529; }
        .changes { background-color: #e7f3ff; padding: 10px; border-left: 4px solid #007bff; margin: 10px 0; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 12px; color: #6c757d; text-align: center; }
        .btn { display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Activity Log Notification</h1>
            <p>{{.CompanyName}}</p>
        </div>
        
        <p>A new activity has been logged in your system:</p>
        
        <div class="activity-details">
            <div class="detail-row">
                <span class="label">Activity:</span>
                <span class="value">{{.ActivityLog.FormattedMessage}}</span>
            </div>
            <div class="detail-row">
                <span class="label">Type:</span>
                <span class="value">{{.ActivityLog.ActivityName}}</span>
            </div>
            <div class="detail-row">
                <span class="label">Object:</span>
                <span class="value">{{.ActivityLog.ObjectName}} ({{.ActivityLog.ObjectID}})</span>
            </div>
            <div class="detail-row">
                <span class="label">Performed by:</span>
                <span class="value">{{.ActivityLog.Actor.Name}} ({{.ActivityLog.Actor.Email}})</span>
            </div>
            <div class="detail-row">
                <span class="label">Time:</span>
                <span class="value">{{.ActivityLog.CreatedAt.Format "2006-01-02 15:04:05 UTC"}}</span>
            </div>
            
            {{if .ActivityLog.Changes}}
            <div class="changes">
                <strong>Changes:</strong><br>
                <pre>{{.ActivityLog.Changes}}</pre>
            </div>
            {{end}}
        </div>
        
        {{if .WebURL}}
        <div style="text-align: center;">
            <a href="{{.WebURL}}/activity-logs/{{.ActivityLog.ID}}" class="btn">View in Dashboard</a>
        </div>
        {{end}}
        
        <div class="footer">
            <p>This is an automated notification from Activity Log Service.</p>
            {{if .UnsubscribeURL}}
            <p><a href="{{.UnsubscribeURL}}">Unsubscribe</a> from these notifications.</p>
            {{end}}
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("activity_log").Parse(activityLogTemplate)
	if err != nil {
		m.logger.WithError(err).Error("Failed to parse activity log email template")
	} else {
		m.templates["activity_log"] = tmpl
	}

	// Summary email template
	summaryTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Daily Activity Summary</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; padding: 20px; border-radius: 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
        .header { background-color: #28a745; color: white; padding: 15px; text-align: center; border-radius: 5px 5px 0 0; margin: -20px -20px 20px -20px; }
        .summary-stats { display: flex; justify-content: space-around; margin: 20px 0; }
        .stat { text-align: center; }
        .stat-number { font-size: 2em; font-weight: bold; color: #007bff; }
        .stat-label { color: #6c757d; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 12px; color: #6c757d; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Daily Activity Summary</h1>
            <p>{{.Date}}</p>
        </div>
        
        <div class="summary-stats">
            <div class="stat">
                <div class="stat-number">{{.TotalActivities}}</div>
                <div class="stat-label">Total Activities</div>
            </div>
            <div class="stat">
                <div class="stat-number">{{.UniqueUsers}}</div>
                <div class="stat-label">Active Users</div>
            </div>
            <div class="stat">
                <div class="stat-number">{{.TopActivity}}</div>
                <div class="stat-label">Most Common Activity</div>
            </div>
        </div>
        
        <div class="footer">
            <p>This is your daily activity summary from Activity Log Service.</p>
        </div>
    </div>
</body>
</html>`

	summaryTmpl, err := template.New("daily_summary").Parse(summaryTemplate)
	if err != nil {
		m.logger.WithError(err).Error("Failed to parse summary email template")
	} else {
		m.templates["daily_summary"] = summaryTmpl
	}
}

func (m *Mailer) SendActivityLogNotification(ctx context.Context, data ActivityLogEmailData) error {
	if len(data.Recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	template, exists := m.templates["activity_log"]
	if !exists {
		return fmt.Errorf("activity log email template not found")
	}

	var body bytes.Buffer
	if err := template.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	subject := data.Subject
	if subject == "" {
		subject = fmt.Sprintf("Activity Log: %s", data.ActivityLog.FormattedMessage)
	}

	return m.sendEmail(ctx, data.Recipients, subject, body.String())
}

func (m *Mailer) SendDailySummary(ctx context.Context, recipients []string, summaryData map[string]interface{}) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	template, exists := m.templates["daily_summary"]
	if !exists {
		return fmt.Errorf("daily summary email template not found")
	}

	var body bytes.Buffer
	if err := template.Execute(&body, summaryData); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	subject := fmt.Sprintf("Daily Activity Summary - %s", time.Now().Format("2006-01-02"))
	return m.sendEmail(ctx, recipients, subject, body.String())
}

func (m *Mailer) sendEmail(ctx context.Context, recipients []string, subject, body string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", recipients...)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body)

	// Add message ID and date headers
	msg.SetHeader("Message-ID", fmt.Sprintf("<%d@activity-log-service>", time.Now().UnixNano()))
	msg.SetHeader("Date", time.Now().Format(time.RFC1123Z))

	if err := m.dialer.DialAndSend(msg); err != nil {
		m.logger.WithError(err).WithFields(logrus.Fields{
			"recipients": recipients,
			"subject":    subject,
		}).Error("Failed to send email")
		return fmt.Errorf("failed to send email: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"recipients": recipients,
		"subject":    subject,
	}).Info("Email sent successfully")

	return nil
}

func (m *Mailer) TestConnection(ctx context.Context) error {
	// Send a test email to verify the connection
	testMsg := gomail.NewMessage()
	testMsg.SetHeader("From", m.from)
	testMsg.SetHeader("To", m.from)
	testMsg.SetHeader("Subject", "Test Email - Activity Log Service")
	testMsg.SetBody("text/plain", "This is a test email to verify the email service configuration.")

	if err := m.dialer.DialAndSend(testMsg); err != nil {
		return fmt.Errorf("failed to send test email: %w", err)
	}

	m.logger.Info("Email service test successful")
	return nil
}
