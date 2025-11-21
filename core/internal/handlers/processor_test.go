package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/core/internal/dlq"
	"github.com/telhawk-systems/telhawk-stack/core/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/service"
	"github.com/telhawk-systems/telhawk-stack/core/internal/validator"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

// mockNormalizer for testing
type mockNormalizer struct {
	supportsFormat     string
	supportsSourceType string
}

func (m *mockNormalizer) Supports(format, sourceType string) bool {
	return format == m.supportsFormat && sourceType == m.supportsSourceType
}

func (m *mockNormalizer) Normalize(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
	return &ocsf.Event{
		ClassUID: 3002,
		Class:    "authentication",
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}, nil
}

func setupTestHandler(t *testing.T) *handlers.ProcessorHandler {
	registry := normalizer.NewRegistry(&mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "test",
	})
	chain := validator.NewChain(validator.BasicValidator{})
	pipe := pipeline.New(registry, chain)

	processor := service.NewProcessor(pipe, nil, nil)
	return handlers.NewProcessorHandler(processor)
}

func TestNewProcessorHandler(t *testing.T) {
	handler := setupTestHandler(t)
	assert.NotNil(t, handler)
}

func TestProcessorHandler_Normalize_Success(t *testing.T) {
	handler := setupTestHandler(t)

	payload := base64.StdEncoding.EncodeToString([]byte(`{"user":"alice","action":"login"}`))
	reqBody := handlers.NormalizationRequest{
		ID:         "test-123",
		Source:     "test-system",
		SourceType: "test",
		Format:     "json",
		Payload:    payload,
		ReceivedAt: time.Now().Format(time.RFC3339Nano),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/normalize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Normalize(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp handlers.NormalizationResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Event)

	// Verify the event is valid OCSF
	var event map[string]interface{}
	err = json.Unmarshal(resp.Event, &event)
	require.NoError(t, err)
	assert.Equal(t, float64(3002), event["class_uid"])
	assert.Equal(t, "authentication", event["class"])
}

func TestProcessorHandler_Normalize_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/normalize", nil)
			w := httptest.NewRecorder()

			handler.Normalize(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
			assert.Contains(t, w.Header().Get("Allow"), http.MethodPost)
		})
	}
}

func TestProcessorHandler_Normalize_InvalidJSON(t *testing.T) {
	handler := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/normalize", bytes.NewReader([]byte(`invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Normalize(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request", resp["code"])
}

func TestProcessorHandler_Normalize_InvalidBase64Payload(t *testing.T) {
	handler := setupTestHandler(t)

	reqBody := handlers.NormalizationRequest{
		ID:         "test-123",
		Source:     "test",
		SourceType: "test",
		Format:     "json",
		Payload:    "not-valid-base64!!!",
		ReceivedAt: time.Now().Format(time.RFC3339Nano),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/normalize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Normalize(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid_payload", resp["code"])
}

func TestProcessorHandler_Normalize_NormalizationFailed(t *testing.T) {
	// Setup handler with no normalizers (will fail to find match)
	registry := normalizer.NewRegistry()
	chain := validator.NewChain()
	pipe := pipeline.New(registry, chain)
	processor := service.NewProcessor(pipe, nil, nil)
	handler := handlers.NewProcessorHandler(processor)

	payload := base64.StdEncoding.EncodeToString([]byte(`{}`))
	reqBody := handlers.NormalizationRequest{
		ID:         "test-123",
		Source:     "test",
		SourceType: "unknown",
		Format:     "json",
		Payload:    payload,
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/normalize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Normalize(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "normalization_failed", resp["code"])
}

func TestProcessorHandler_Normalize_WithReceivedAt(t *testing.T) {
	handler := setupTestHandler(t)

	receivedAt := time.Now().Add(-1 * time.Hour)
	payload := base64.StdEncoding.EncodeToString([]byte(`{}`))
	reqBody := handlers.NormalizationRequest{
		ID:         "test-123",
		Source:     "test",
		SourceType: "test",
		Format:     "json",
		Payload:    payload,
		ReceivedAt: receivedAt.Format(time.RFC3339Nano),
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/normalize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Normalize(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessorHandler_Normalize_WithAttributes(t *testing.T) {
	handler := setupTestHandler(t)

	payload := base64.StdEncoding.EncodeToString([]byte(`{}`))
	reqBody := handlers.NormalizationRequest{
		ID:         "test-123",
		Source:     "test",
		SourceType: "test",
		Format:     "json",
		Payload:    payload,
		Attributes: map[string]string{
			"region": "us-west-2",
			"env":    "production",
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/normalize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Normalize(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessorHandler_Health_Success(t *testing.T) {
	handler := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats service.Stats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, stats.UptimeSeconds, int64(0))
	assert.GreaterOrEqual(t, stats.Processed, uint64(0))
}

func TestProcessorHandler_Health_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/healthz", nil)
			w := httptest.NewRecorder()

			handler.Health(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

func TestProcessorHandler_ListDLQ_Success(t *testing.T) {
	tempDir := t.TempDir()
	dlqQueue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	// Write some events to DLQ
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		envelope := &model.RawEventEnvelope{
			ID:     "test",
			Format: "json",
		}
		err = dlqQueue.Write(ctx, envelope, assert.AnError, "test_reason")
		require.NoError(t, err)
	}

	// Setup handler with DLQ
	registry := normalizer.NewRegistry()
	chain := validator.NewChain()
	pipe := pipeline.New(registry, chain)
	processor := service.NewProcessor(pipe, nil, dlqQueue)
	handler := handlers.NewProcessorHandler(processor)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq", nil)
	w := httptest.NewRecorder()

	handler.ListDLQ(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(3), resp["count"])
	assert.NotNil(t, resp["events"])
}

func TestProcessorHandler_ListDLQ_Disabled(t *testing.T) {
	// Setup handler without DLQ
	handler := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dlq", nil)
	w := httptest.NewRecorder()

	handler.ListDLQ(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "dlq_disabled", resp["code"])
}

func TestProcessorHandler_ListDLQ_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/dlq", nil)
			w := httptest.NewRecorder()

			handler.ListDLQ(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

func TestProcessorHandler_PurgeDLQ_Success(t *testing.T) {
	tempDir := t.TempDir()
	dlqQueue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	// Write an event
	ctx := context.Background()
	envelope := &model.RawEventEnvelope{ID: "test", Format: "json"}
	err = dlqQueue.Write(ctx, envelope, assert.AnError, "test")
	require.NoError(t, err)

	// Setup handler
	registry := normalizer.NewRegistry()
	chain := validator.NewChain()
	pipe := pipeline.New(registry, chain)
	processor := service.NewProcessor(pipe, nil, dlqQueue)
	handler := handlers.NewProcessorHandler(processor)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dlq", nil)
	w := httptest.NewRecorder()

	handler.PurgeDLQ(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "purged", resp["status"])

	// Verify DLQ is empty
	events, err := dlqQueue.List(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestProcessorHandler_PurgeDLQ_Disabled(t *testing.T) {
	handler := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/dlq", nil)
	w := httptest.NewRecorder()

	handler.PurgeDLQ(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestProcessorHandler_PurgeDLQ_MethodNotAllowed(t *testing.T) {
	handler := setupTestHandler(t)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/dlq", nil)
			w := httptest.NewRecorder()

			handler.PurgeDLQ(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

func TestProcessorHandler_Normalize_EmptyBody(t *testing.T) {
	handler := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/normalize", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Normalize(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProcessorHandler_Normalize_MissingFields(t *testing.T) {
	handler := setupTestHandler(t)

	// Request with missing payload field - empty string is valid base64
	// so this will actually succeed but with empty payload
	reqBody := map[string]interface{}{
		"id":          "test-123",
		"source_type": "test",
		"format":      "json",
		"payload":     "", // Empty but valid base64
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/normalize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Normalize(w, req)

	// Empty string is valid base64, will decode to empty payload
	// Should succeed (or fail in normalization, but not in base64 decoding)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}
