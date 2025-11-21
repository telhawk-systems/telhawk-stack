package normalizer_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

func TestHECNormalizer_Supports(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	testCases := []struct {
		name       string
		format     string
		sourceType string
		expected   bool
	}{
		{
			name:       "json format with hec sourceType",
			format:     "json",
			sourceType: "hec",
			expected:   true,
		},
		{
			name:       "json format with non-hec sourceType",
			format:     "json",
			sourceType: "syslog",
			expected:   false,
		},
		{
			name:       "xml format with hec sourceType",
			format:     "xml",
			sourceType: "hec",
			expected:   false,
		},
		{
			name:       "empty format",
			format:     "",
			sourceType: "hec",
			expected:   false,
		},
		{
			name:       "empty sourceType",
			format:     "json",
			sourceType: "",
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := norm.Supports(tc.format, tc.sourceType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHECNormalizer_Normalize_BasicEvent(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"event": "User logged in",
		"host":  "web-server-01",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		ID:         "test-001",
		Format:     "json",
		SourceType: "hec",
		Source:     "splunk-forwarder",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Verify OCSF base event structure
	assert.Equal(t, 0, event.ClassUID, "HEC events should have class_uid 0 (base_event)")
	assert.Equal(t, "base_event", event.Class)
	assert.Equal(t, "other", event.Category)
	assert.Equal(t, ocsf.CategoryOther, event.CategoryUID)

	// Verify metadata
	assert.Equal(t, "TelHawk Stack", event.Metadata.Product.Name)
	assert.Equal(t, "TelHawk Systems", event.Metadata.Product.Vendor)
	assert.Equal(t, "1.1.0", event.Metadata.Version)

	// Verify properties
	assert.Equal(t, "splunk-forwarder", event.Properties["source"])
	assert.Equal(t, "hec", event.Properties["source_type"])

	// Verify raw data preservation
	assert.Equal(t, "json", event.Raw.Format)
	assert.NotNil(t, event.Raw.Data)

	// Verify timing
	assert.False(t, event.Time.IsZero())
	assert.False(t, event.ObservedTime.IsZero())
}

func TestHECNormalizer_Normalize_WithTimestamp(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	// Unix timestamp with nanoseconds
	unixTime := 1699999999.123456
	payload := map[string]interface{}{
		"time":  unixTime,
		"event": "Test event with timestamp",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Verify time was converted from Unix timestamp
	expectedTime := time.Unix(int64(unixTime), int64((unixTime-float64(int64(unixTime)))*1e9))

	// Allow 1 second tolerance for time conversion
	timeDiff := event.Time.Sub(expectedTime)
	assert.True(t, timeDiff >= -time.Second && timeDiff <= time.Second,
		"Time should be within 1 second of expected: expected=%v, got=%v, diff=%v",
		expectedTime, event.Time, timeDiff)
}

func TestHECNormalizer_Normalize_WithoutTimestamp(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"event": "Test event without timestamp",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	receivedAt := time.Now()
	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: receivedAt,
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// When no time in payload, should use ReceivedAt
	assert.Equal(t, receivedAt, event.Time)
	assert.Equal(t, receivedAt, event.ObservedTime)
}

func TestHECNormalizer_Normalize_InvalidJSON(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    []byte(`{invalid json`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	assert.Error(t, err)
	assert.Nil(t, event)
	assert.Contains(t, err.Error(), "decode hec payload")
}

func TestHECNormalizer_Normalize_EmptyPayload(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Should still create a valid base event
	assert.Equal(t, 0, event.ClassUID)
	assert.Equal(t, "base_event", event.Class)
}

func TestHECNormalizer_Normalize_ComplexPayload(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"time":  1699999999.0,
		"event": "Complex event",
		"fields": map[string]interface{}{
			"user":     "alice",
			"action":   "login",
			"ip":       "192.168.1.100",
			"severity": "high",
		},
		"metadata": map[string]interface{}{
			"version": "2.0",
			"source":  "application",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Verify raw data contains the entire payload
	assert.NotNil(t, event.Raw.Data)
	rawData, ok := event.Raw.Data.(map[string]interface{})
	require.True(t, ok)

	// Check nested fields are preserved
	fields, ok := rawData["fields"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "alice", fields["user"])
	assert.Equal(t, "login", fields["action"])
}

func TestHECNormalizer_Normalize_ActivityField(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"event": "test",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Activity should reflect the ingestion source
	assert.Equal(t, "ingest:hec", event.Activity)
}

func TestHECNormalizer_Normalize_SeverityAndStatus(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"event": "test",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Default severity and status for HEC events
	assert.Equal(t, ocsf.SeverityUnknown, event.SeverityID)
	assert.Equal(t, "Unknown", event.Severity)
	assert.Equal(t, ocsf.StatusUnknown, event.StatusID)
	assert.Equal(t, "Unknown", event.Status)
}

func TestHECNormalizer_Normalize_TypeUID(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"event": "test",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// TypeUID should be computed correctly
	expectedTypeUID := ocsf.ComputeTypeUID(event.CategoryUID, event.ClassUID, event.ActivityID)
	assert.Equal(t, expectedTypeUID, event.TypeUID)
}

func TestHECNormalizer_Normalize_Attributes(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"event": "test",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
		Attributes: map[string]string{
			"region": "us-west-2",
			"env":    "production",
		},
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Attributes aren't automatically copied to properties in HEC normalizer
	// but envelope is processed without error
	assert.NotNil(t, event.Properties)
}

func TestHECNormalizer_Normalize_TimeConversion_EdgeCases(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	testCases := []struct {
		name    string
		time    interface{}
		wantErr bool
	}{
		{
			name:    "valid float time",
			time:    1699999999.123,
			wantErr: false,
		},
		{
			name:    "integer time",
			time:    1699999999,
			wantErr: false,
		},
		{
			name:    "zero time",
			time:    0.0,
			wantErr: false,
		},
		{
			name:    "negative time",
			time:    -1.0,
			wantErr: false,
		},
		{
			name:    "very large time",
			time:    9999999999.0,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"time":  tc.time,
				"event": "test",
			}

			payloadBytes, err := json.Marshal(payload)
			require.NoError(t, err)

			envelope := &model.RawEventEnvelope{
				Format:     "json",
				SourceType: "hec",
				Source:     "test",
				Payload:    payloadBytes,
				ReceivedAt: time.Now(),
			}

			ctx := context.Background()
			event, err := norm.Normalize(ctx, envelope)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, event)
				assert.False(t, event.Time.IsZero())
			}
		})
	}
}

func TestHECNormalizer_Normalize_PropertiesAlwaysSet(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"event": "test",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test-source",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Properties should always be set
	require.NotNil(t, event.Properties)
	assert.Equal(t, "test-source", event.Properties["source"])
	assert.Equal(t, "hec", event.Properties["source_type"])
}

func TestHECNormalizer_Normalize_RawDataPreserved(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	originalPayload := map[string]interface{}{
		"event": "test event",
		"custom_field": map[string]interface{}{
			"nested": "value",
		},
	}

	payloadBytes, err := json.Marshal(originalPayload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := norm.Normalize(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)

	// Raw data should preserve the original payload
	assert.Equal(t, "json", event.Raw.Format)
	require.NotNil(t, event.Raw.Data)

	rawData, ok := event.Raw.Data.(map[string]interface{})
	require.True(t, ok, "raw data should be a map")

	assert.Equal(t, "test event", rawData["event"])
	customField, ok := rawData["custom_field"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", customField["nested"])
}

func TestHECNormalizer_Normalize_ContextPassing(t *testing.T) {
	norm := normalizer.HECNormalizer{}

	payload := map[string]interface{}{
		"event": "test",
	}

	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "hec",
		Source:     "test",
		Payload:    payloadBytes,
		ReceivedAt: time.Now(),
	}

	// Test with cancelled context (HECNormalizer doesn't use context currently)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event, err := norm.Normalize(ctx, envelope)

	// Should succeed even with cancelled context since HEC normalizer doesn't check it
	require.NoError(t, err)
	require.NotNil(t, event)
}
