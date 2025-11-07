package notification

import (
	"context"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

func TestWebhookChannel(t *testing.T) {
	t.Skip("requires external network access")
	channel := NewWebhookChannel("https://httpbin.org/post", 5*time.Second)
	
	if channel.Type() != "webhook" {
		t.Errorf("expected type 'webhook', got '%s'", channel.Type())
	}
	
	alert := &models.Alert{
		ID:          "test-alert",
		Name:        "Test Alert",
		Description: "Test description",
		Query:       "test query",
		Severity:    "high",
	}
	
	results := []map[string]interface{}{
		{"message": "test event 1"},
		{"message": "test event 2"},
	}
	
	ctx := context.Background()
	if err := channel.Send(ctx, alert, results); err != nil {
		t.Fatalf("expected webhook send to succeed: %v", err)
	}
}

func TestSlackChannel(t *testing.T) {
	t.Skip("requires external network access")
	channel := NewSlackChannel("https://httpbin.org/post", 5*time.Second)
	
	if channel.Type() != "slack" {
		t.Errorf("expected type 'slack', got '%s'", channel.Type())
	}
	
	alert := &models.Alert{
		ID:          "test-alert",
		Name:        "Test Alert",
		Description: "Test description",
		Query:       "test query",
		Severity:    "critical",
	}
	
	results := []map[string]interface{}{
		{"message": "test event"},
	}
	
	ctx := context.Background()
	if err := channel.Send(ctx, alert, results); err != nil {
		t.Fatalf("expected slack send to succeed: %v", err)
	}
}

func TestLogChannel(t *testing.T) {
	logged := false
	logFunc := func(format string, v ...interface{}) {
		logged = true
	}
	
	channel := NewLogChannel(logFunc)
	
	if channel.Type() != "log" {
		t.Errorf("expected type 'log', got '%s'", channel.Type())
	}
	
	alert := &models.Alert{
		ID:       "test-alert",
		Name:     "Test Alert",
		Severity: "medium",
	}
	
	ctx := context.Background()
	if err := channel.Send(ctx, alert, nil); err != nil {
		t.Fatalf("expected log send to succeed: %v", err)
	}
	
	if !logged {
		t.Error("expected log function to be called")
	}
}

func TestMultiChannel(t *testing.T) {
	logCount := 0
	logFunc := func(format string, v ...interface{}) {
		logCount++
	}
	
	log1 := NewLogChannel(logFunc)
	log2 := NewLogChannel(logFunc)
	
	multi := NewMultiChannel(log1, log2)
	
	if multi.Type() != "multi" {
		t.Errorf("expected type 'multi', got '%s'", multi.Type())
	}
	
	alert := &models.Alert{
		ID:       "test-alert",
		Name:     "Test Alert",
		Severity: "low",
	}
	
	ctx := context.Background()
	if err := multi.Send(ctx, alert, nil); err != nil {
		t.Fatalf("expected multi-channel send to succeed: %v", err)
	}
	
	if logCount != 2 {
		t.Errorf("expected 2 log calls, got %d", logCount)
	}
}

func TestSlackSeverityColors(t *testing.T) {
	channel := &SlackChannel{}
	
	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "#8B0000"},
		{"high", "#FF0000"},
		{"medium", "#FFA500"},
		{"low", "#FFFF00"},
		{"info", "#0000FF"},
		{"unknown", "#808080"},
	}
	
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			color := channel.severityColor(tt.severity)
			if color != tt.expected {
				t.Errorf("expected color %s for severity %s, got %s", tt.expected, tt.severity, color)
			}
		})
	}
}
