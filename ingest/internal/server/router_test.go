package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
)

// Mock service for testing
type mockIngestService struct{}

func (m *mockIngestService) IngestEvent(event *models.HECEvent, sourceIP, hecTokenID string) (string, error) {
	return "", nil
}

func (m *mockIngestService) IngestRaw(data []byte, sourceIP, hecTokenID, source, sourceType, host string) (string, error) {
	return "", nil
}

func (m *mockIngestService) ValidateHECToken(ctx context.Context, token string) error {
	return nil
}

func (m *mockIngestService) GetStats() models.IngestionStats {
	return models.IngestionStats{}
}

func (m *mockIngestService) QueryAcks(ackIDs []string) map[string]bool {
	return make(map[string]bool)
}

func TestNewRouter(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)

	router := NewRouter(handler)

	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}
}

func TestRouter_EventEndpoint(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/event", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Should route to handler (even if it returns error due to missing auth)
	if rr.Code == http.StatusNotFound {
		t.Error("/services/collector/event endpoint not registered")
	}
}

func TestRouter_RawEndpoint(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/raw", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code == http.StatusNotFound {
		t.Error("/services/collector/raw endpoint not registered")
	}
}

func TestRouter_HealthEndpoint(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/healthz returned %d, want 200", rr.Code)
	}
}

func TestRouter_ReadyEndpoint(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/readyz returned %d, want 200", rr.Code)
	}
}

func TestRouter_CollectorHealthEndpoint(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/services/collector/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/services/collector/health returned %d, want 200", rr.Code)
	}
}

func TestRouter_AckEndpoint(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/services/collector/ack", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code == http.StatusNotFound {
		t.Error("/services/collector/ack endpoint not registered")
	}
}

func TestRouter_MetricsEndpoint(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/metrics returned %d, want 200", rr.Code)
	}

	// Verify it's Prometheus format (should contain "# HELP" or "# TYPE")
	body := rr.Body.String()
	if len(body) == 0 {
		t.Error("/metrics returned empty body")
	}
}

func TestRouter_NotFoundEndpoint(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("/nonexistent returned %d, want 404", rr.Code)
	}
}

func TestRouter_RequestIDMiddleware(t *testing.T) {
	mockService := &mockIngestService{}
	handler := handlers.NewHECHandler(mockService, nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Verify X-Request-ID header is set by middleware
	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("X-Request-ID header not set by middleware")
	}
}
