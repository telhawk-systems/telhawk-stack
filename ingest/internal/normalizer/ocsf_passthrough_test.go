package normalizer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
)

func TestOCSFPassthroughNormalizer_Supports(t *testing.T) {
	normalizer := OCSFPassthroughNormalizer{}

	tests := []struct {
		name       string
		format     string
		sourceType string
		expected   bool
	}{
		{"OCSF auth", "json", "ocsf:authentication", true},
		{"OCSF network", "json", "ocsf:network_activity", true},
		{"OCSF process", "json", "ocsf:process_activity", true},
		{"OCSF file", "json", "ocsf:file_activity", true},
		{"OCSF uppercase", "json", "OCSF:detection", true},
		{"HEC format", "json", "hec", false},
		{"Non-OCSF", "json", "syslog", false},
		{"Wrong format", "xml", "ocsf:auth", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Supports(tt.format, tt.sourceType)
			if result != tt.expected {
				t.Errorf("Supports(%q, %q) = %v, expected %v",
					tt.format, tt.sourceType, result, tt.expected)
			}
		})
	}
}

func TestOCSFPassthroughNormalizer_NormalizeHECEnvelope(t *testing.T) {
	normalizer := OCSFPassthroughNormalizer{}

	// Test event wrapped in HEC envelope format
	hecPayload := map[string]interface{}{
		"time":       1699999999.5,
		"sourcetype": "ocsf:authentication",
		"event": map[string]interface{}{
			"class_uid":     3002,
			"class_name":    "Authentication",
			"activity_id":   1,
			"activity_name": "login",
			"severity_id":   1,
			"user": map[string]interface{}{
				"name": "testuser",
				"uid":  "test-uid-123",
			},
			"message": "User login attempt",
		},
	}

	payload, _ := json.Marshal(hecPayload)
	envelope := &models.RawEventEnvelope{
		Format:     "json",
		SourceType: "ocsf:authentication",
		Source:     "test-source",
		Payload:    payload,
		ReceivedAt: time.Now(),
	}

	event, err := normalizer.Normalize(context.Background(), envelope)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Verify the event was extracted from the envelope
	if event.ClassUID != 3002 {
		t.Errorf("Expected ClassUID 3002, got %d", event.ClassUID)
	}
	if event.ActivityID != 1 {
		t.Errorf("Expected ActivityID 1, got %d", event.ActivityID)
	}
	if event.SeverityID != 1 {
		t.Errorf("Expected SeverityID 1, got %d", event.SeverityID)
	}

	// Verify properties were set
	if event.Properties["source_type"] != "ocsf:authentication" {
		t.Errorf("Expected source_type to be set in properties")
	}
}

func TestOCSFPassthroughNormalizer_NormalizeDirectOCSF(t *testing.T) {
	normalizer := OCSFPassthroughNormalizer{}

	// Test event already in OCSF format (not wrapped)
	ocsfPayload := map[string]interface{}{
		"class_uid":     4001,
		"class_name":    "Network Activity",
		"activity_id":   2,
		"activity_name": "open",
		"severity_id":   1,
		"src_endpoint": map[string]interface{}{
			"ip":   "192.168.1.100",
			"port": 54321,
		},
		"dst_endpoint": map[string]interface{}{
			"ip":   "10.0.0.50",
			"port": 443,
		},
	}

	payload, _ := json.Marshal(ocsfPayload)
	envelope := &models.RawEventEnvelope{
		Format:     "json",
		SourceType: "ocsf:network_activity",
		Source:     "network-tap",
		Payload:    payload,
		ReceivedAt: time.Now(),
	}

	event, err := normalizer.Normalize(context.Background(), envelope)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Verify the OCSF fields were preserved
	if event.ClassUID != 4001 {
		t.Errorf("Expected ClassUID 4001, got %d", event.ClassUID)
	}
	if event.ActivityID != 2 {
		t.Errorf("Expected ActivityID 2, got %d", event.ActivityID)
	}

	// Verify envelope metadata was added
	if event.Properties["source"] != "network-tap" {
		t.Errorf("Expected source to be set from envelope")
	}
}

func TestOCSFPassthroughNormalizer_TimeConversion(t *testing.T) {
	normalizer := OCSFPassthroughNormalizer{}

	// Test Unix timestamp conversion
	unixTime := float64(1699999999.123456789)
	hecPayload := map[string]interface{}{
		"event": map[string]interface{}{
			"class_uid":  2004,
			"class_name": "Detection Finding",
			"time":       unixTime,
		},
		"sourcetype": "ocsf:detection_finding",
	}

	payload, _ := json.Marshal(hecPayload)
	envelope := &models.RawEventEnvelope{
		Format:     "json",
		SourceType: "ocsf:detection_finding",
		Source:     "ids",
		Payload:    payload,
		ReceivedAt: time.Now(),
	}

	event, err := normalizer.Normalize(context.Background(), envelope)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Verify time was converted and can be parsed
	if event.Time.IsZero() {
		t.Error("Time should not be zero after conversion")
	}

	// The time should be close to the Unix timestamp
	expectedTime := time.Unix(int64(unixTime), int64((unixTime-float64(int64(unixTime)))*1e9))
	timeDiff := event.Time.Sub(expectedTime)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Time conversion off by more than 1 second: expected %v, got %v",
			expectedTime, event.Time)
	}
}

func TestOCSFPassthroughNormalizer_InvalidPayload(t *testing.T) {
	normalizer := OCSFPassthroughNormalizer{}

	tests := []struct {
		name    string
		payload string
	}{
		{"Invalid JSON", "{not valid json}"},
		{"Empty payload", ""},
		{"Non-object", `["array", "not", "object"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope := &models.RawEventEnvelope{
				Format:     "json",
				SourceType: "ocsf:test",
				Source:     "test",
				Payload:    []byte(tt.payload),
				ReceivedAt: time.Now(),
			}

			_, err := normalizer.Normalize(context.Background(), envelope)
			if err == nil {
				t.Error("Expected error for invalid payload, got nil")
			}
		})
	}
}

func TestOCSFPassthroughNormalizer_PreservesRawData(t *testing.T) {
	normalizer := OCSFPassthroughNormalizer{}

	originalPayload := map[string]interface{}{
		"event": map[string]interface{}{
			"class_uid":  1007,
			"class_name": "Process Activity",
			"process": map[string]interface{}{
				"pid":      12345,
				"name":     "nginx",
				"cmd_line": "/usr/sbin/nginx -g daemon off;",
			},
		},
		"sourcetype": "ocsf:process_activity",
		"time":       1699999999.0,
	}

	payload, _ := json.Marshal(originalPayload)
	envelope := &models.RawEventEnvelope{
		Format:     "json",
		SourceType: "ocsf:process_activity",
		Source:     "agent",
		Payload:    payload,
		ReceivedAt: time.Now(),
	}

	event, err := normalizer.Normalize(context.Background(), envelope)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Verify raw data is preserved
	if event.Raw.Format != "json" {
		t.Errorf("Expected raw format 'json', got %s", event.Raw.Format)
	}
	if event.Raw.Data == nil {
		t.Error("Raw data should be preserved")
	}
}

func TestOCSFPassthroughNormalizer_PropertiesSet(t *testing.T) {
	normalizer := OCSFPassthroughNormalizer{}

	hecPayload := map[string]interface{}{
		"event": map[string]interface{}{
			"class_uid": 4006,
		},
		"sourcetype": "ocsf:file_activity",
	}

	payload, _ := json.Marshal(hecPayload)
	envelope := &models.RawEventEnvelope{
		Format:     "json",
		SourceType: "ocsf:file_activity",
		Source:     "file-monitor",
		Payload:    payload,
		ReceivedAt: time.Now(),
	}

	event, err := normalizer.Normalize(context.Background(), envelope)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	// Verify required properties are set
	if event.Properties == nil {
		t.Fatal("Properties should not be nil")
	}
	if event.Properties["source"] != "file-monitor" {
		t.Errorf("Expected source 'file-monitor', got %v", event.Properties["source"])
	}
	if event.Properties["source_type"] != "ocsf:file_activity" {
		t.Errorf("Expected source_type 'ocsf:file_activity', got %v",
			event.Properties["source_type"])
	}

	// Verify ObservedTime was set
	if event.ObservedTime.IsZero() {
		t.Error("ObservedTime should be set from envelope")
	}
}
