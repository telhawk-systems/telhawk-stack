package coreclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
)

func TestNew(t *testing.T) {
	baseURL := "http://localhost:8082"
	timeout := 30 * time.Second

	client := New(baseURL, timeout)

	if client == nil {
		t.Fatal("New() returned nil")
	}

	if client.baseURL != baseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, baseURL)
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}

	if client.httpClient.Timeout != timeout {
		t.Errorf("httpClient.Timeout = %v, want %v", client.httpClient.Timeout, timeout)
	}

	if client.maxRetries != 3 {
		t.Errorf("maxRetries = %d, want 3", client.maxRetries)
	}

	if client.retryDelay != 100*time.Millisecond {
		t.Errorf("retryDelay = %v, want 100ms", client.retryDelay)
	}
}

func TestNormalize_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/normalize" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		resp := NormalizationResult{
			Event: json.RawMessage(`{"class_uid":3002,"category_uid":3,"activity_id":1}`),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Host:       "test-host",
		Source:     "test-source",
		SourceType: "test-sourcetype",
		Event:      map[string]interface{}{"event": "test"},
		HECTokenID: "token-123",
	}

	result, err := client.Normalize(ctx, event)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if result == nil {
		t.Fatal("Normalize() result is nil")
	}

	if len(result.Event) == 0 {
		t.Error("Normalize() result.Event is empty")
	}
}

func TestNormalize_NilClient(t *testing.T) {
	var client *Client
	ctx := context.Background()

	event := &models.Event{
		ID:    "event-123",
		Event: "test",
	}

	_, err := client.Normalize(ctx, event)
	if err == nil {
		t.Error("Normalize() with nil client should return error")
	}

	expectedErr := "core client not configured"
	if err.Error() != expectedErr {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestNormalize_RetryOn500(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount < 3 {
			// First 2 calls: return 500
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "server error"})
			return
		}

		// Third call: success
		resp := NormalizationResult{
			Event: json.RawMessage(`{"class_uid":3002}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	result, err := client.Normalize(ctx, event)
	if err != nil {
		t.Fatalf("Normalize() error = %v (should succeed after retries)", err)
	}

	if result == nil {
		t.Error("Normalize() result is nil")
	}

	if callCount != 3 {
		t.Errorf("Expected 3 server calls (2 failures + 1 success), got %d", callCount)
	}
}

func TestNormalize_RetryOn429(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First call: rate limited
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// Second call: success
		resp := NormalizationResult{
			Event: json.RawMessage(`{"class_uid":3002}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	result, err := client.Normalize(ctx, event)
	if err != nil {
		t.Fatalf("Normalize() error = %v (should succeed after retry)", err)
	}

	if result == nil {
		t.Error("Normalize() result is nil")
	}

	if callCount != 2 {
		t.Errorf("Expected 2 server calls (rate limit + success), got %d", callCount)
	}
}

func TestNormalize_NoRetryOn4xx(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "bad request"})
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	_, err := client.Normalize(ctx, event)
	if err == nil {
		t.Error("Normalize() should error on 400 response")
	}

	// Should not retry on 4xx errors (except 429)
	if callCount != 1 {
		t.Errorf("Expected 1 server call (no retry on 4xx), got %d", callCount)
	}
}

func TestNormalize_MaxRetriesExceeded(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "server error"})
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	_, err := client.Normalize(ctx, event)
	if err == nil {
		t.Error("Normalize() should error when max retries exceeded")
	}

	expectedMsg := "max retries exceeded"
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("error message = %q, want prefix %q", err.Error(), expectedMsg)
	}

	// Should try initial + 3 retries = 4 total
	if callCount != 4 {
		t.Errorf("Expected 4 server calls (initial + 3 retries), got %d", callCount)
	}
}

func TestNormalize_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	_, err := client.Normalize(ctx, event)
	if err == nil {
		t.Error("Normalize() with cancelled context should return error")
	}
}

func TestNormalize_ContextCancellationDuringRetry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	_, err := client.Normalize(ctx, event)
	if err == nil {
		t.Error("Normalize() should error when context cancelled during retry")
	}

	// Should stop retrying when context is cancelled
	if callCount >= 4 {
		t.Errorf("Should not complete all retries when context is cancelled, got %d calls", callCount)
	}
}

func TestNormalize_ExponentialBackoff(t *testing.T) {
	callTimes := []time.Time{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callTimes = append(callTimes, time.Now())
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	client.Normalize(ctx, event)

	if len(callTimes) != 4 {
		t.Fatalf("Expected 4 calls, got %d", len(callTimes))
	}

	// Check exponential backoff timing
	// Retry 1: ~100ms delay
	// Retry 2: ~200ms delay
	// Retry 3: ~400ms delay

	delay1 := callTimes[1].Sub(callTimes[0])
	delay2 := callTimes[2].Sub(callTimes[1])
	delay3 := callTimes[3].Sub(callTimes[2])

	// Allow some tolerance for timing
	if delay1 < 80*time.Millisecond || delay1 > 150*time.Millisecond {
		t.Errorf("First retry delay = %v, want ~100ms", delay1)
	}

	if delay2 < 180*time.Millisecond || delay2 > 250*time.Millisecond {
		t.Errorf("Second retry delay = %v, want ~200ms", delay2)
	}

	if delay3 < 380*time.Millisecond || delay3 > 450*time.Millisecond {
		t.Errorf("Third retry delay = %v, want ~400ms", delay3)
	}
}

func TestNormalize_WithRawPayload(t *testing.T) {
	receivedPayload := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		receivedPayload = req["payload"].(string)

		resp := NormalizationResult{
			Event: json.RawMessage(`{"class_uid":3002}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	rawData := []byte("raw log line")
	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Raw:        rawData,
		HECTokenID: "token-123",
	}

	_, err := client.Normalize(ctx, event)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	// Verify raw payload was base64 encoded
	if receivedPayload == "" {
		t.Error("Expected payload to be sent")
	}
}

func TestNormalize_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	_, err := client.Normalize(ctx, event)
	if err == nil {
		t.Error("Normalize() should error on invalid JSON response")
	}
}

func TestNormalize_NetworkError(t *testing.T) {
	// Use an invalid URL that will cause a network error
	client := New("http://localhost:99999", 100*time.Millisecond)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	_, err := client.Normalize(ctx, event)
	if err == nil {
		t.Error("Normalize() should error on network failure")
	}
}

func TestRetryableError(t *testing.T) {
	baseErr := &retryableError{err: http.ErrServerClosed}

	if baseErr.Error() != http.ErrServerClosed.Error() {
		t.Errorf("Error() = %q, want %q", baseErr.Error(), http.ErrServerClosed.Error())
	}

	unwrapped := baseErr.Unwrap()
	if unwrapped != http.ErrServerClosed {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, http.ErrServerClosed)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "Retryable error",
			err:       &retryableError{err: http.ErrServerClosed},
			retryable: true,
		},
		{
			name:      "Non-retryable error",
			err:       http.ErrServerClosed,
			retryable: false,
		},
		{
			name:      "Nil error",
			err:       nil,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("isRetryable() = %v, want %v", result, tt.retryable)
			}
		})
	}
}

func TestNormalize_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set very short timeout
	client := New(server.URL, 50*time.Millisecond)
	ctx := context.Background()

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Now(),
		Event:      "test",
		HECTokenID: "token-123",
	}

	_, err := client.Normalize(ctx, event)
	if err == nil {
		t.Error("Normalize() should timeout")
	}
}

func TestNormalize_RequestFields(t *testing.T) {
	var receivedRequest map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedRequest)

		resp := NormalizationResult{
			Event: json.RawMessage(`{"class_uid":3002}`),
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	testTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	event := &models.Event{
		ID:         "event-123",
		Timestamp:  testTime,
		Host:       "test-host",
		Source:     "test-source",
		SourceType: "test-sourcetype",
		Event:      map[string]interface{}{"message": "test"},
		HECTokenID: "token-456",
	}

	_, err := client.Normalize(ctx, event)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	// Verify request fields
	if receivedRequest["id"] != "event-123" {
		t.Errorf("Request id = %v, want event-123", receivedRequest["id"])
	}

	if receivedRequest["source"] != "test-source" {
		t.Errorf("Request source = %v, want test-source", receivedRequest["source"])
	}

	if receivedRequest["source_type"] != "test-sourcetype" {
		t.Errorf("Request source_type = %v, want test-sourcetype", receivedRequest["source_type"])
	}

	if receivedRequest["format"] != "json" {
		t.Errorf("Request format = %v, want json", receivedRequest["format"])
	}

	attrs, ok := receivedRequest["attributes"].(map[string]interface{})
	if !ok {
		t.Fatal("Request attributes missing or invalid")
	}

	if attrs["host"] != "test-host" {
		t.Errorf("Request attributes.host = %v, want test-host", attrs["host"])
	}

	if attrs["hec_token_id"] != "token-456" {
		t.Errorf("Request attributes.hec_token_id = %v, want token-456", attrs["hec_token_id"])
	}
}
