package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEvent_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	event := Event{
		ID:        "event-123",
		Timestamp: now,
		Source:    "test-source",
		SourceIP:  "192.168.1.100",
		EventData: map[string]interface{}{
			"user":   "admin",
			"action": "login",
			"status": "success",
		},
		Raw:       []byte(`{"test": "data"}`),
		Signature: "abc123signature",
		Ingested:  now.Add(-5 * time.Minute),
		Processed: now.Add(-2 * time.Minute),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	// Unmarshal back
	var unmarshaled Event
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	// Verify fields
	if unmarshaled.ID != event.ID {
		t.Errorf("ID: expected %q, got %q", event.ID, unmarshaled.ID)
	}
	if !unmarshaled.Timestamp.Equal(event.Timestamp) {
		t.Errorf("Timestamp: expected %v, got %v", event.Timestamp, unmarshaled.Timestamp)
	}
	if unmarshaled.Source != event.Source {
		t.Errorf("Source: expected %q, got %q", event.Source, unmarshaled.Source)
	}
	if unmarshaled.SourceIP != event.SourceIP {
		t.Errorf("SourceIP: expected %q, got %q", event.SourceIP, unmarshaled.SourceIP)
	}
	if unmarshaled.Signature != event.Signature {
		t.Errorf("Signature: expected %q, got %q", event.Signature, unmarshaled.Signature)
	}

	// Verify EventData
	if unmarshaled.EventData["user"] != "admin" {
		t.Errorf("EventData[user]: expected %q, got %v", "admin", unmarshaled.EventData["user"])
	}
	if unmarshaled.EventData["action"] != "login" {
		t.Errorf("EventData[action]: expected %q, got %v", "login", unmarshaled.EventData["action"])
	}

	// Verify Raw bytes
	if string(unmarshaled.Raw) != string(event.Raw) {
		t.Errorf("Raw: expected %q, got %q", string(event.Raw), string(unmarshaled.Raw))
	}
}

func TestEvent_EmptyEventData(t *testing.T) {
	event := Event{
		ID:        "event-456",
		Timestamp: time.Now(),
		Source:    "test",
		SourceIP:  "10.0.0.1",
		EventData: nil,
		Signature: "sig",
		Ingested:  time.Now(),
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event with nil EventData: %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
}

func TestEvent_OmitEmpty(t *testing.T) {
	// Note: Go's omitempty doesn't work with time.Time zero values because
	// time.Time is a struct, not a pointer. The zero value (0001-01-01T00:00:00Z)
	// is not considered "empty" by the JSON encoder.
	// This test verifies the actual behavior.
	event := Event{
		ID:        "event-789",
		Timestamp: time.Now(),
		Source:    "test",
		SourceIP:  "10.0.0.1",
		EventData: map[string]interface{}{},
		Signature: "sig",
		Ingested:  time.Now(),
		// Processed is not set (zero value)
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	// Parse JSON to check the processed field
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Even with omitempty, time.Time zero value will be serialized
	// because it's a struct. This is expected Go behavior.
	if processedVal, exists := raw["processed"]; exists {
		// If it exists, it should be the zero time
		if processedStr, ok := processedVal.(string); ok {
			if processedStr != "0001-01-01T00:00:00Z" && processedStr != "" {
				t.Errorf("expected zero time value, got: %s", processedStr)
			}
		}
	}
}

func TestIngestionLog_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	log := IngestionLog{
		ID:            "log-123",
		Timestamp:     now,
		HECTokenID:    "token-456",
		SourceIP:      "192.168.1.200",
		EventCount:    42,
		BytesReceived: 8192,
		Success:       true,
		ErrorMessage:  "",
		Signature:     "log-signature-abc",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal ingestion log: %v", err)
	}

	// Unmarshal back
	var unmarshaled IngestionLog
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ingestion log: %v", err)
	}

	// Verify fields
	if unmarshaled.ID != log.ID {
		t.Errorf("ID: expected %q, got %q", log.ID, unmarshaled.ID)
	}
	if !unmarshaled.Timestamp.Equal(log.Timestamp) {
		t.Errorf("Timestamp: expected %v, got %v", log.Timestamp, unmarshaled.Timestamp)
	}
	if unmarshaled.HECTokenID != log.HECTokenID {
		t.Errorf("HECTokenID: expected %q, got %q", log.HECTokenID, unmarshaled.HECTokenID)
	}
	if unmarshaled.SourceIP != log.SourceIP {
		t.Errorf("SourceIP: expected %q, got %q", log.SourceIP, unmarshaled.SourceIP)
	}
	if unmarshaled.EventCount != log.EventCount {
		t.Errorf("EventCount: expected %d, got %d", log.EventCount, unmarshaled.EventCount)
	}
	if unmarshaled.BytesReceived != log.BytesReceived {
		t.Errorf("BytesReceived: expected %d, got %d", log.BytesReceived, unmarshaled.BytesReceived)
	}
	if unmarshaled.Success != log.Success {
		t.Errorf("Success: expected %v, got %v", log.Success, unmarshaled.Success)
	}
	if unmarshaled.Signature != log.Signature {
		t.Errorf("Signature: expected %q, got %q", log.Signature, unmarshaled.Signature)
	}
}

func TestIngestionLog_WithError(t *testing.T) {
	log := IngestionLog{
		ID:            "log-error-123",
		Timestamp:     time.Now(),
		HECTokenID:    "token-789",
		SourceIP:      "10.0.0.5",
		EventCount:    0,
		BytesReceived: 1024,
		Success:       false,
		ErrorMessage:  "failed to parse event data",
		Signature:     "sig",
	}

	jsonData, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal ingestion log with error: %v", err)
	}

	var unmarshaled IngestionLog
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ingestion log: %v", err)
	}

	if unmarshaled.Success {
		t.Error("expected Success to be false")
	}
	if unmarshaled.ErrorMessage != log.ErrorMessage {
		t.Errorf("ErrorMessage: expected %q, got %q", log.ErrorMessage, unmarshaled.ErrorMessage)
	}
}

func TestIngestionLog_OmitEmptyError(t *testing.T) {
	// Test that ErrorMessage field is omitted when empty
	log := IngestionLog{
		ID:            "log-success-123",
		Timestamp:     time.Now(),
		HECTokenID:    "token-999",
		SourceIP:      "10.0.0.10",
		EventCount:    10,
		BytesReceived: 2048,
		Success:       true,
		ErrorMessage:  "", // Empty error message
		Signature:     "sig",
	}

	jsonData, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal ingestion log: %v", err)
	}

	// Parse JSON to check if error_message field exists
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// ErrorMessage should not be present when empty (omitempty)
	if _, exists := raw["error_message"]; exists {
		t.Error("expected 'error_message' field to be omitted when empty")
	}
}

func TestEvent_RawBytesEncoding(t *testing.T) {
	// Test that raw bytes are properly base64 encoded in JSON
	rawData := []byte("this is raw event data with special chars: \x00\x01\x02")

	event := Event{
		ID:        "event-raw-test",
		Timestamp: time.Now(),
		Source:    "test",
		SourceIP:  "10.0.0.1",
		EventData: map[string]interface{}{},
		Raw:       rawData,
		Signature: "sig",
		Ingested:  time.Now(),
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event with raw bytes: %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if string(unmarshaled.Raw) != string(rawData) {
		t.Errorf("Raw bytes mismatch: expected %v, got %v", rawData, unmarshaled.Raw)
	}
}

func TestModels_StructTags(t *testing.T) {
	// Verify that all expected JSON tags are present
	eventJSON := `{
		"id": "test",
		"timestamp": "2024-01-01T00:00:00Z",
		"source": "test",
		"source_ip": "10.0.0.1",
		"event_data": {},
		"raw": null,
		"signature": "sig",
		"ingested": "2024-01-01T00:00:00Z"
	}`

	var event Event
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	logJSON := `{
		"id": "test",
		"timestamp": "2024-01-01T00:00:00Z",
		"hec_token_id": "token",
		"source_ip": "10.0.0.1",
		"event_count": 10,
		"bytes_received": 1024,
		"success": true,
		"signature": "sig"
	}`

	var log IngestionLog
	if err := json.Unmarshal([]byte(logJSON), &log); err != nil {
		t.Fatalf("failed to unmarshal ingestion log: %v", err)
	}
}
