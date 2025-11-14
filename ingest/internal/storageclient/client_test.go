package storageclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	baseURL := "http://localhost:8083"
	timeout := 15 * time.Second

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
}

func TestIngest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ingest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var req IngestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if len(req.Events) != 2 {
			t.Errorf("expected 2 events, got %d", len(req.Events))
		}

		resp := IngestResponse{
			Indexed: 2,
			Failed:  0,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1", "message": "test event 1"},
		{"id": "event-2", "message": "test event 2"},
	}

	result, err := client.Ingest(ctx, events)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if result == nil {
		t.Fatal("Ingest() result is nil")
	}

	if result.Indexed != 2 {
		t.Errorf("Indexed = %d, want 2", result.Indexed)
	}

	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}
}

func TestIngest_PartialSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := IngestResponse{
			Indexed: 1,
			Failed:  1,
			Errors:  []string{"validation error on event 2"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPartialContent)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1", "message": "test event 1"},
		{"id": "event-2", "invalid": "data"},
	}

	result, err := client.Ingest(ctx, events)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if result.Indexed != 1 {
		t.Errorf("Indexed = %d, want 1", result.Indexed)
	}

	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}

	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}
}

func TestIngest_NilClient(t *testing.T) {
	var client *Client
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() with nil client should return error")
	}

	expectedErr := "storage client not configured"
	if err.Error() != expectedErr {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestIngest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "internal server error"})
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() should error on server error")
	}
}

func TestIngest_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid request"})
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() should error on bad request")
	}
}

func TestIngest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() with cancelled context should return error")
	}
}

func TestIngest_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set very short timeout
	client := New(server.URL, 50*time.Millisecond)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() should timeout")
	}
}

func TestIngest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() should error on invalid JSON response")
	}
}

func TestIngest_EmptyEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req IngestRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Events) != 0 {
			t.Errorf("expected 0 events, got %d", len(req.Events))
		}

		resp := IngestResponse{
			Indexed: 0,
			Failed:  0,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{}

	result, err := client.Ingest(ctx, events)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if result.Indexed != 0 {
		t.Errorf("Indexed = %d, want 0", result.Indexed)
	}
}

func TestIngest_NilEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req IngestRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp := IngestResponse{
			Indexed: 0,
			Failed:  0,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	result, err := client.Ingest(ctx, nil)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if result.Indexed != 0 {
		t.Errorf("Indexed = %d, want 0", result.Indexed)
	}
}

func TestIngest_LargeEventBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req IngestRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp := IngestResponse{
			Indexed: len(req.Events),
			Failed:  0,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	// Create 100 events
	events := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		events[i] = map[string]interface{}{
			"id":      i,
			"message": "test event",
		}
	}

	result, err := client.Ingest(ctx, events)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if result.Indexed != 100 {
		t.Errorf("Indexed = %d, want 100", result.Indexed)
	}
}

func TestIngest_NetworkError(t *testing.T) {
	// Use an invalid URL that will cause a network error
	client := New("http://localhost:99999", 100*time.Millisecond)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() should error on network failure")
	}
}

func TestIngest_MalformedURL(t *testing.T) {
	// Client with invalid base URL
	client := New("http://[invalid", 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() should error with malformed URL")
	}
}

func TestIngest_ErrorsInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := IngestResponse{
			Indexed: 8,
			Failed:  2,
			Errors: []string{
				"event 3: validation failed",
				"event 7: duplicate ID",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPartialContent)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := make([]map[string]interface{}, 10)
	for i := 0; i < 10; i++ {
		events[i] = map[string]interface{}{"id": i}
	}

	result, err := client.Ingest(ctx, events)
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if result.Indexed != 8 {
		t.Errorf("Indexed = %d, want 8", result.Indexed)
	}

	if result.Failed != 2 {
		t.Errorf("Failed = %d, want 2", result.Failed)
	}

	if len(result.Errors) != 2 {
		t.Errorf("len(Errors) = %d, want 2", len(result.Errors))
	}
}

func TestIngest_ServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"message": "service unavailable"})
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() should error on service unavailable")
	}
}

func TestIngest_NoErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		// No body
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second)
	ctx := context.Background()

	events := []map[string]interface{}{
		{"id": "event-1"},
	}

	_, err := client.Ingest(ctx, events)
	if err == nil {
		t.Error("Ingest() should error even without error message in response")
	}
}
