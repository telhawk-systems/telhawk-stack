package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

// Channel defines the interface for alert notification delivery.
type Channel interface {
	Send(ctx context.Context, alert *models.Alert, results []map[string]interface{}) error
	Type() string
}

// WebhookChannel sends alert notifications via HTTP POST.
type WebhookChannel struct {
	URL     string
	Timeout time.Duration
	client  *http.Client
}

// NewWebhookChannel creates a webhook notification channel.
func NewWebhookChannel(url string, timeout time.Duration) *WebhookChannel {
	return &WebhookChannel{
		URL:     url,
		Timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (w *WebhookChannel) Type() string {
	return "webhook"
}

func (w *WebhookChannel) Send(ctx context.Context, alert *models.Alert, results []map[string]interface{}) error {
	payload := map[string]interface{}{
		"alert_id":     alert.ID,
		"alert_name":   alert.Name,
		"severity":     alert.Severity,
		"description":  alert.Description,
		"query":        alert.Query,
		"result_count": len(results),
		"results":      results,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TelHawk-Query/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// SlackChannel sends alert notifications to Slack via webhook.
type SlackChannel struct {
	WebhookURL string
	Timeout    time.Duration
	client     *http.Client
}

// NewSlackChannel creates a Slack notification channel.
func NewSlackChannel(webhookURL string, timeout time.Duration) *SlackChannel {
	return &SlackChannel{
		WebhookURL: webhookURL,
		Timeout:    timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (s *SlackChannel) Type() string {
	return "slack"
}

func (s *SlackChannel) Send(ctx context.Context, alert *models.Alert, results []map[string]interface{}) error {
	color := s.severityColor(alert.Severity)

	payload := map[string]interface{}{
		"text": fmt.Sprintf("🚨 Alert: %s", alert.Name),
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"fields": []map[string]interface{}{
					{
						"title": "Alert Name",
						"value": alert.Name,
						"short": true,
					},
					{
						"title": "Severity",
						"value": alert.Severity,
						"short": true,
					},
					{
						"title": "Results",
						"value": fmt.Sprintf("%d events matched", len(results)),
						"short": true,
					},
					{
						"title": "Query",
						"value": fmt.Sprintf("```%s```", alert.Query),
						"short": false,
					},
				},
				"footer": "TelHawk Alert",
				"ts":     time.Now().Unix(),
			},
		},
	}

	if alert.Description != "" {
		payload["attachments"].([]map[string]interface{})[0]["text"] = alert.Description
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *SlackChannel) severityColor(severity string) string {
	switch severity {
	case "critical":
		return "#8B0000"
	case "high":
		return "#FF0000"
	case "medium":
		return "#FFA500"
	case "low":
		return "#FFFF00"
	case "info":
		return "#0000FF"
	default:
		return "#808080"
	}
}

// LogChannel writes alert notifications to logs (for testing/debugging).
type LogChannel struct {
	logger func(format string, v ...interface{})
}

// NewLogChannel creates a log-based notification channel.
func NewLogChannel(logger func(format string, v ...interface{})) *LogChannel {
	return &LogChannel{logger: logger}
}

func (l *LogChannel) Type() string {
	return "log"
}

func (l *LogChannel) Send(ctx context.Context, alert *models.Alert, results []map[string]interface{}) error {
	l.logger("ALERT TRIGGERED: %s (id=%s, severity=%s, results=%d)",
		alert.Name, alert.ID, alert.Severity, len(results))
	return nil
}

// MultiChannel sends notifications to multiple channels.
type MultiChannel struct {
	channels []Channel
}

// NewMultiChannel creates a notification channel that fans out to multiple channels.
func NewMultiChannel(channels ...Channel) *MultiChannel {
	return &MultiChannel{channels: channels}
}

func (m *MultiChannel) Type() string {
	return "multi"
}

func (m *MultiChannel) Send(ctx context.Context, alert *models.Alert, results []map[string]interface{}) error {
	var lastErr error
	successCount := 0

	for _, ch := range m.channels {
		if err := ch.Send(ctx, alert, results); err != nil {
			lastErr = fmt.Errorf("%s channel failed: %w", ch.Type(), err)
		} else {
			successCount++
		}
	}

	if successCount == 0 && len(m.channels) > 0 {
		return fmt.Errorf("all notification channels failed: %w", lastErr)
	}

	return nil
}
