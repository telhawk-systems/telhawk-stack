package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
)

// Mock service for testing
type mockIngestService struct {
	validateTokenErr error
	ingestEventAckID string
	ingestEventErr   error
	ingestRawAckID   string
	ingestRawErr     error
}

func (m *mockIngestService) IngestEvent(event *models.HECEvent, sourceIP, token string) (string, error) {
	return m.ingestEventAckID, m.ingestEventErr
}

func (m *mockIngestService) IngestRaw(data []byte, sourceIP, token, source, sourceType, host string) (string, error) {
	return m.ingestRawAckID, m.ingestRawErr
}

func (m *mockIngestService) ValidateHECToken(ctx context.Context, token string) error {
	return m.validateTokenErr
}

func (m *mockIngestService) GetStats() models.IngestionStats {
	return models.IngestionStats{}
}

func (m *mockIngestService) QueryAcks(ackIDs []string) map[string]bool {
	result := make(map[string]bool)
	for _, id := range ackIDs {
		result[id] = true
	}
	return result
}

func TestHandleEvent_WithAck(t *testing.T) {
	mockService := &mockIngestService{
		ingestEventAckID: "test-ack-id-123",
	}

	handler := NewHECHandler(mockService, nil)

	event := map[string]interface{}{
		"event": "test event",
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("X-Splunk-Request-Channel", "channel-123")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response models.HECResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.AckID != "test-ack-id-123" {
		t.Errorf("Expected ackId 'test-ack-id-123', got '%s'", response.AckID)
	}

	if response.Code != 0 {
		t.Errorf("Expected code 0, got %d", response.Code)
	}

	if response.Text != "Success" {
		t.Errorf("Expected text 'Success', got '%s'", response.Text)
	}
}

func TestHandleEvent_WithoutAck(t *testing.T) {
	mockService := &mockIngestService{
		ingestEventAckID: "test-ack-id-123",
	}

	handler := NewHECHandler(mockService, nil)

	event := map[string]interface{}{
		"event": "test event",
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Telhawk test-token")
	// No X-Splunk-Request-Channel header
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response models.HECResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.AckID != "" {
		t.Errorf("Expected no ackId, got '%s'", response.AckID)
	}

	if response.Code != 0 {
		t.Errorf("Expected code 0, got %d", response.Code)
	}

	if response.Text != "Success" {
		t.Errorf("Expected text 'Success', got '%s'", response.Text)
	}
}

func TestHandleRaw_WithAck(t *testing.T) {
	mockService := &mockIngestService{
		ingestRawAckID: "raw-ack-id-456",
	}

	handler := NewHECHandler(mockService, nil)

	rawData := []byte("test raw log line")

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", bytes.NewReader(rawData))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("X-Splunk-Request-Channel", "channel-456")

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response models.HECResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.AckID != "raw-ack-id-456" {
		t.Errorf("Expected ackId 'raw-ack-id-456', got '%s'", response.AckID)
	}
}

func TestAckQuery(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	reqBody := map[string]interface{}{
		"acks": []string{"ack-1", "ack-2"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/ack", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Ack(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	acks, ok := response["acks"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'acks' field in response")
	}

	if acks["ack-1"] != true {
		t.Errorf("Expected ack-1 to be true")
	}

	if acks["ack-2"] != true {
		t.Errorf("Expected ack-2 to be true")
	}
}

func TestHandleEvent_InvalidJSON(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	// Invalid JSON
	body := []byte("{invalid json}")

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("Expected non-200 status for invalid JSON")
	}
}

func TestHandleEvent_EmptyBody(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader([]byte{}))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("Expected non-200 status for empty body")
	}
}

func TestHandleEvent_ServiceError(t *testing.T) {
	mockService := &mockIngestService{
		ingestEventErr: fmt.Errorf("service unavailable"),
	}
	handler := NewHECHandler(mockService, nil)

	event := map[string]interface{}{
		"event": "test event",
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("Expected non-200 status when service returns error")
	}
}

func TestHandleRaw_EmptyBody(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", bytes.NewReader([]byte{}))
	req.Header.Set("Authorization", "Telhawk test-token")

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("Expected non-200 status for empty raw body")
	}
}

func TestHandleRaw_ServiceError(t *testing.T) {
	mockService := &mockIngestService{
		ingestRawErr: fmt.Errorf("service unavailable"),
	}
	handler := NewHECHandler(mockService, nil)

	rawData := []byte("test raw log line")

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", bytes.NewReader(rawData))
	req.Header.Set("Authorization", "Telhawk test-token")

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("Expected non-200 status when service returns error")
	}
}

func TestHealth(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestReady(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handler.Ready(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check if status field exists
	status, hasStatus := response["status"]
	if !hasStatus {
		t.Fatal("Expected 'status' field in response")
	}

	// Status should be "ready" (not "failed")
	if status != "ready" && status != "healthy" {
		t.Errorf("Expected status 'ready' or 'healthy', got %v", status)
	}
}

func TestAck_InvalidJSON(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	body := []byte("{invalid json}")

	req := httptest.NewRequest(http.MethodPost, "/services/collector/ack", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Ack(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("Expected non-200 status for invalid JSON")
	}
}

func TestHandleEvent_MissingToken(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	event := map[string]interface{}{"event": "test"}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestHandleEvent_TokenValidationFailed(t *testing.T) {
	mockService := &mockIngestService{
		validateTokenErr: fmt.Errorf("invalid token"),
	}
	handler := NewHECHandler(mockService, nil)

	event := map[string]interface{}{"event": "test"}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Telhawk bad-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestHandleEvent_WrongMethod(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/services/collector/event", nil)

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rr.Code)
	}
}

func TestHandleRaw_WrongMethod(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/services/collector/raw", nil)

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rr.Code)
	}
}

func TestHandleRaw_MissingToken(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", bytes.NewReader([]byte("raw data")))

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestHandleRaw_TokenValidationFailed(t *testing.T) {
	mockService := &mockIngestService{
		validateTokenErr: fmt.Errorf("invalid token"),
	}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", bytes.NewReader([]byte("raw data")))
	req.Header.Set("Authorization", "Telhawk bad-token")

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestHandleRaw_WithQueryParams(t *testing.T) {
	mockService := &mockIngestService{
		ingestRawAckID: "raw-ack-456",
	}
	handler := NewHECHandler(mockService, nil)

	rawData := []byte("test raw log line")

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw?source=test-source&sourcetype=test-type&host=test-host", bytes.NewReader(rawData))
	req.Header.Set("Authorization", "Telhawk test-token")

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestHandleRaw_WithHeaders(t *testing.T) {
	mockService := &mockIngestService{
		ingestRawAckID: "raw-ack-789",
	}
	handler := NewHECHandler(mockService, nil)

	rawData := []byte("test raw log line")

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", bytes.NewReader(rawData))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("X-Splunk-Request-Source", "header-source")
	req.Header.Set("X-Splunk-Request-Sourcetype", "header-type")
	req.Header.Set("X-Splunk-Request-Host", "header-host")

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestHandleEvent_NDJSONBatch(t *testing.T) {
	mockService := &mockIngestService{
		ingestEventAckID: "batch-ack-123",
	}
	handler := NewHECHandler(mockService, nil)

	ndjson := `{"event":"event 1","source":"test"}
{"event":"event 2","source":"test"}
{"event":"event 3","source":"test"}`

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader([]byte(ndjson)))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestHandleEvent_NDJSONBatchWithEmptyLines(t *testing.T) {
	mockService := &mockIngestService{
		ingestEventAckID: "batch-ack-456",
	}
	handler := NewHECHandler(mockService, nil)

	ndjson := `{"event":"event 1"}

{"event":"event 2"}

{"event":"event 3"}`

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader([]byte(ndjson)))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestHandleEvent_NDJSONInvalidLine(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	ndjson := `{"event":"event 1"}
{invalid json}
{"event":"event 3"}`

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader([]byte(ndjson)))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("Expected non-200 status for invalid NDJSON line")
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")

	ip := getClientIP(req)
	if ip != "1.2.3.4" {
		t.Errorf("getClientIP() = %q, want %q", ip, "1.2.3.4")
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "9.10.11.12")

	ip := getClientIP(req)
	if ip != "9.10.11.12" {
		t.Errorf("getClientIP() = %q, want %q", ip, "9.10.11.12")
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "13.14.15.16:12345"

	ip := getClientIP(req)
	if ip != "13.14.15.16:12345" {
		t.Errorf("getClientIP() = %q, want %q", ip, "13.14.15.16:12345")
	}
}

func TestGetClientIP_Precedence(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-IP", "5.6.7.8")
	req.RemoteAddr = "9.10.11.12:12345"

	ip := getClientIP(req)
	if ip != "1.2.3.4" {
		t.Errorf("getClientIP() = %q, want %q", ip, "1.2.3.4")
	}
}

func TestAck_GetMethod(t *testing.T) {
	mockService := &mockIngestService{}
	handler := NewHECHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/services/collector/ack", nil)
	rr := httptest.NewRecorder()

	handler.Ack(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

type mockRateLimiter struct {
	allowFunc func(ctx context.Context, key string) (bool, error)
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if m.allowFunc != nil {
		return m.allowFunc(ctx, key)
	}
	return true, nil
}

func (m *mockRateLimiter) Close() error {
	return nil
}

func TestHandleEvent_RateLimitedByIP(t *testing.T) {
	mockService := &mockIngestService{}
	rateLimiter := &mockRateLimiter{
		allowFunc: func(ctx context.Context, key string) (bool, error) {
			if strings.HasPrefix(key, "ip:") {
				return false, nil
			}
			return true, nil
		},
	}
	handler := NewHECHandler(mockService, rateLimiter)

	event := map[string]interface{}{"event": "test"}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rr.Code)
	}
}

func TestHandleEvent_RateLimitedByToken(t *testing.T) {
	mockService := &mockIngestService{}
	rateLimiter := &mockRateLimiter{
		allowFunc: func(ctx context.Context, key string) (bool, error) {
			if strings.HasPrefix(key, "token:") {
				return false, nil
			}
			return true, nil
		},
	}
	handler := NewHECHandler(mockService, rateLimiter)

	event := map[string]interface{}{"event": "test"}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rr.Code)
	}
}

func TestHandleEvent_RateLimitError(t *testing.T) {
	mockService := &mockIngestService{}
	rateLimiter := &mockRateLimiter{
		allowFunc: func(ctx context.Context, key string) (bool, error) {
			return false, fmt.Errorf("rate limiter error")
		},
	}
	handler := NewHECHandler(mockService, rateLimiter)

	event := map[string]interface{}{"event": "test"}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", bytes.NewReader(body))
	req.Header.Set("Authorization", "Telhawk test-token")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.HandleEvent(rr, req)

	if rr.Code == http.StatusTooManyRequests {
		t.Error("Should not return 429 when rate limiter has an error")
	}
}

func TestHandleRaw_RateLimitedByIP(t *testing.T) {
	mockService := &mockIngestService{}
	rateLimiter := &mockRateLimiter{
		allowFunc: func(ctx context.Context, key string) (bool, error) {
			if strings.HasPrefix(key, "ip:") {
				return false, nil
			}
			return true, nil
		},
	}
	handler := NewHECHandler(mockService, rateLimiter)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", bytes.NewReader([]byte("raw data")))
	req.Header.Set("Authorization", "Telhawk test-token")

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rr.Code)
	}
}

func TestHandleRaw_RateLimitedByToken(t *testing.T) {
	mockService := &mockIngestService{}
	rateLimiter := &mockRateLimiter{
		allowFunc: func(ctx context.Context, key string) (bool, error) {
			if strings.HasPrefix(key, "token:") {
				return false, nil
			}
			return true, nil
		},
	}
	handler := NewHECHandler(mockService, rateLimiter)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", bytes.NewReader([]byte("raw data")))
	req.Header.Set("Authorization", "Telhawk test-token")

	rr := httptest.NewRecorder()
	handler.HandleRaw(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rr.Code)
	}
}
