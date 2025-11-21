package notification

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

func TestWebhookChannel_PayloadStructure(t *testing.T) {
	// Create a test server to capture the webhook payload
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "TelHawk-Query/1.0", r.Header.Get("User-Agent"))
		assert.Equal(t, http.MethodPost, r.Method)

		// Capture payload
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel := NewWebhookChannel(server.URL, 5*time.Second)
	alert := &models.Alert{
		ID:          "test-alert-123",
		Name:        "Test Alert",
		Description: "Test description",
		Query:       "severity:high",
		Severity:    "high",
	}

	results := []map[string]interface{}{
		{"message": "event 1", "severity": "high"},
		{"message": "event 2", "severity": "high"},
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, results)

	require.NoError(t, err)
	require.NotNil(t, receivedPayload)

	// Verify payload structure
	assert.Equal(t, "test-alert-123", receivedPayload["alert_id"])
	assert.Equal(t, "Test Alert", receivedPayload["alert_name"])
	assert.Equal(t, "high", receivedPayload["severity"])
	assert.Equal(t, "Test description", receivedPayload["description"])
	assert.Equal(t, "severity:high", receivedPayload["query"])
	assert.Equal(t, float64(2), receivedPayload["result_count"])
	assert.NotEmpty(t, receivedPayload["timestamp"])
	assert.NotNil(t, receivedPayload["results"])
}

func TestWebhookChannel_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse int
		expectError    bool
	}{
		{"200 OK", http.StatusOK, false},
		{"201 Created", http.StatusCreated, false},
		{"204 No Content", http.StatusNoContent, false},
		{"400 Bad Request", http.StatusBadRequest, true},
		{"401 Unauthorized", http.StatusUnauthorized, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverResponse)
			}))
			defer server.Close()

			channel := NewWebhookChannel(server.URL, 5*time.Second)
			alert := &models.Alert{
				ID:       "test-alert",
				Name:     "Test",
				Severity: "high",
			}

			ctx := context.Background()
			err := channel.Send(ctx, alert, nil)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "webhook returned status")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookChannel_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set very short timeout
	channel := NewWebhookChannel(server.URL, 50*time.Millisecond)
	alert := &models.Alert{
		ID:       "test-alert",
		Name:     "Test",
		Severity: "high",
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send webhook")
}

func TestWebhookChannel_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if context was cancelled
		select {
		case <-r.Context().Done():
			return
		case <-time.After(1 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	channel := NewWebhookChannel(server.URL, 5*time.Second)
	alert := &models.Alert{
		ID:       "test-alert",
		Name:     "Test",
		Severity: "high",
	}

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := channel.Send(ctx, alert, nil)

	assert.Error(t, err)
}

func TestSlackChannel_PayloadStructure(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel := NewSlackChannel(server.URL, 5*time.Second)
	alert := &models.Alert{
		ID:          "test-alert",
		Name:        "Critical Alert",
		Description: "This is critical",
		Query:       "severity:critical",
		Severity:    "critical",
	}

	results := []map[string]interface{}{
		{"message": "event 1"},
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, results)

	require.NoError(t, err)
	require.NotNil(t, receivedPayload)

	// Verify Slack-specific structure
	assert.Contains(t, receivedPayload["text"], "Critical Alert")
	assert.Contains(t, receivedPayload, "attachments")

	attachments := receivedPayload["attachments"].([]interface{})
	require.Len(t, attachments, 1)

	attachment := attachments[0].(map[string]interface{})
	assert.Equal(t, "#8B0000", attachment["color"]) // critical color
	assert.Contains(t, attachment, "fields")
	assert.Equal(t, "TelHawk Alert", attachment["footer"])
	assert.NotNil(t, attachment["ts"])
}

func TestSlackChannel_AllSeverityColors(t *testing.T) {
	tests := []struct {
		severity      string
		expectedColor string
	}{
		{"critical", "#8B0000"},
		{"high", "#FF0000"},
		{"medium", "#FFA500"},
		{"low", "#FFFF00"},
		{"info", "#0000FF"},
		{"unknown", "#808080"},
		{"", "#808080"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			var receivedColor string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]interface{}
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &payload)

				attachments := payload["attachments"].([]interface{})
				attachment := attachments[0].(map[string]interface{})
				receivedColor = attachment["color"].(string)

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			channel := NewSlackChannel(server.URL, 5*time.Second)
			alert := &models.Alert{
				ID:       "test-alert",
				Name:     "Test",
				Severity: tt.severity,
			}

			ctx := context.Background()
			err := channel.Send(ctx, alert, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedColor, receivedColor)
		})
	}
}

func TestSlackChannel_WithDescription(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel := NewSlackChannel(server.URL, 5*time.Second)
	alert := &models.Alert{
		ID:          "test-alert",
		Name:        "Test Alert",
		Description: "This is a detailed description",
		Severity:    "high",
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, nil)

	require.NoError(t, err)

	attachments := receivedPayload["attachments"].([]interface{})
	attachment := attachments[0].(map[string]interface{})

	assert.Equal(t, "This is a detailed description", attachment["text"])
}

func TestLogChannel_FormatsCorrectly(t *testing.T) {
	var loggedMessage string
	var loggedArgs []interface{}

	logFunc := func(format string, v ...interface{}) {
		loggedMessage = format
		loggedArgs = v
	}

	channel := NewLogChannel(logFunc)
	alert := &models.Alert{
		ID:       "alert-123",
		Name:     "Test Alert",
		Severity: "high",
	}

	results := []map[string]interface{}{
		{"event": "1"},
		{"event": "2"},
		{"event": "3"},
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, results)

	require.NoError(t, err)
	assert.NotEmpty(t, loggedMessage)
	assert.Len(t, loggedArgs, 4) // name, id, severity, result count

	// Check format string contains expected placeholders
	assert.Contains(t, loggedMessage, "ALERT TRIGGERED")
	assert.Contains(t, loggedMessage, "%s")
	assert.Contains(t, loggedMessage, "%d")

	// Check args
	assert.Equal(t, "Test Alert", loggedArgs[0])
	assert.Equal(t, "alert-123", loggedArgs[1])
	assert.Equal(t, "high", loggedArgs[2])
	assert.Equal(t, 3, loggedArgs[3])
}

func TestMultiChannel_AllSucceed(t *testing.T) {
	callCount := 0
	logFunc := func(format string, v ...interface{}) {
		callCount++
	}

	ch1 := NewLogChannel(logFunc)
	ch2 := NewLogChannel(logFunc)
	ch3 := NewLogChannel(logFunc)

	multi := NewMultiChannel(ch1, ch2, ch3)

	alert := &models.Alert{
		ID:       "test",
		Name:     "Test",
		Severity: "high",
	}

	ctx := context.Background()
	err := multi.Send(ctx, alert, nil)

	require.NoError(t, err)
	assert.Equal(t, 3, callCount, "all three channels should be called")
}

func TestMultiChannel_PartialFailure(t *testing.T) {
	successCount := 0
	logFunc := func(format string, v ...interface{}) {
		successCount++
	}

	successChannel := NewLogChannel(logFunc)

	// Create a failing channel (invalid URL)
	failChannel := NewWebhookChannel("http://invalid-url-that-does-not-exist", 1*time.Second)

	multi := NewMultiChannel(successChannel, failChannel, successChannel)

	alert := &models.Alert{
		ID:       "test",
		Name:     "Test",
		Severity: "high",
	}

	ctx := context.Background()
	err := multi.Send(ctx, alert, nil)

	// Should succeed because at least one channel succeeded
	assert.NoError(t, err)
	assert.Equal(t, 2, successCount, "two success channels should be called")
}

func TestMultiChannel_AllFail(t *testing.T) {
	failChannel1 := NewWebhookChannel("http://invalid-url-1", 1*time.Second)
	failChannel2 := NewWebhookChannel("http://invalid-url-2", 1*time.Second)

	multi := NewMultiChannel(failChannel1, failChannel2)

	alert := &models.Alert{
		ID:       "test",
		Name:     "Test",
		Severity: "high",
	}

	ctx := context.Background()
	err := multi.Send(ctx, alert, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all notification channels failed")
}

func TestMultiChannel_Empty(t *testing.T) {
	multi := NewMultiChannel()

	alert := &models.Alert{
		ID:       "test",
		Name:     "Test",
		Severity: "high",
	}

	ctx := context.Background()
	err := multi.Send(ctx, alert, nil)

	// Empty multi-channel should not fail
	assert.NoError(t, err)
}

func TestChannelType(t *testing.T) {
	tests := []struct {
		name         string
		channel      Channel
		expectedType string
	}{
		{
			name:         "webhook",
			channel:      NewWebhookChannel("http://example.com", 5*time.Second),
			expectedType: "webhook",
		},
		{
			name:         "slack",
			channel:      NewSlackChannel("http://example.com", 5*time.Second),
			expectedType: "slack",
		},
		{
			name:         "log",
			channel:      NewLogChannel(func(string, ...interface{}) {}),
			expectedType: "log",
		},
		{
			name:         "multi",
			channel:      NewMultiChannel(),
			expectedType: "multi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedType, tt.channel.Type())
		})
	}
}

func TestWebhookChannel_InvalidURL(t *testing.T) {
	channel := NewWebhookChannel("://invalid-url", 5*time.Second)
	alert := &models.Alert{
		ID:       "test",
		Name:     "Test",
		Severity: "high",
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, nil)

	assert.Error(t, err)
}

func TestWebhookChannel_LargePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify we can receive large payloads
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Greater(t, len(body), 10000) // Should be large
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel := NewWebhookChannel(server.URL, 5*time.Second)
	alert := &models.Alert{
		ID:          "test",
		Name:        "Test",
		Description: strings.Repeat("Large description ", 1000),
		Severity:    "high",
	}

	// Create large result set
	results := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		results[i] = map[string]interface{}{
			"message": strings.Repeat("event data ", 100),
			"index":   i,
		}
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, results)

	assert.NoError(t, err)
}

func TestSlackChannel_FieldMapping(t *testing.T) {
	var receivedFields []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &payload)

		attachments := payload["attachments"].([]interface{})
		attachment := attachments[0].(map[string]interface{})
		fields := attachment["fields"].([]interface{})

		for _, f := range fields {
			receivedFields = append(receivedFields, f.(map[string]interface{}))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channel := NewSlackChannel(server.URL, 5*time.Second)
	alert := &models.Alert{
		ID:       "test",
		Name:     "Test Alert",
		Query:    "severity:high",
		Severity: "high",
	}

	results := []map[string]interface{}{
		{"event": "1"},
		{"event": "2"},
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, results)

	require.NoError(t, err)
	require.Len(t, receivedFields, 4) // Alert Name, Severity, Results, Query

	// Verify field structure
	for _, field := range receivedFields {
		assert.Contains(t, field, "title")
		assert.Contains(t, field, "value")
		assert.Contains(t, field, "short")
	}
}

func TestMultiChannel_ErrorPropagation(t *testing.T) {
	successCount := 0
	logFunc := func(format string, v ...interface{}) {
		successCount++
	}

	// One success, one failure
	success := NewLogChannel(logFunc)
	fail := NewWebhookChannel("http://invalid-url", 1*time.Second)

	multi := NewMultiChannel(success, fail)

	alert := &models.Alert{
		ID:       "test",
		Name:     "Test",
		Severity: "high",
	}

	ctx := context.Background()
	err := multi.Send(ctx, alert, nil)

	// Should not return error because one succeeded
	assert.NoError(t, err)
	assert.Equal(t, 1, successCount)
}

func TestWebhookChannel_RetryNotImplemented(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	channel := NewWebhookChannel(server.URL, 5*time.Second)
	alert := &models.Alert{
		ID:       "test",
		Name:     "Test",
		Severity: "high",
	}

	ctx := context.Background()
	err := channel.Send(ctx, alert, nil)

	assert.Error(t, err)
	// Current implementation doesn't retry, so count should be 1
	assert.Equal(t, 1, attemptCount)
}
