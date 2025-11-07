package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func (m *mockIngestService) ValidateHECToken(token string) error {
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
