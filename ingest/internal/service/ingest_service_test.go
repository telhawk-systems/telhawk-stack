package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/authclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/coreclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/storageclient"
)

// Mock implementations

type mockNormalizationClient struct {
	normalizeFunc func(ctx context.Context, event *models.Event) (*coreclient.NormalizationResult, error)
}

func (m *mockNormalizationClient) Normalize(ctx context.Context, event *models.Event) (*coreclient.NormalizationResult, error) {
	if m.normalizeFunc != nil {
		return m.normalizeFunc(ctx, event)
	}
	// Default: return simple normalized event
	normalized := map[string]interface{}{
		"id":        event.ID,
		"timestamp": event.Timestamp,
		"event":     event.Event,
	}
	data, _ := json.Marshal(normalized)
	return &coreclient.NormalizationResult{Event: data}, nil
}

type mockStorageClient struct {
	ingestFunc func(ctx context.Context, events []map[string]interface{}) (*storageclient.IngestResponse, error)
}

func (m *mockStorageClient) Ingest(ctx context.Context, events []map[string]interface{}) (*storageclient.IngestResponse, error) {
	if m.ingestFunc != nil {
		return m.ingestFunc(ctx, events)
	}
	return &storageclient.IngestResponse{Indexed: len(events), Failed: 0}, nil
}

type mockAuthClient struct {
	validateFunc func(ctx context.Context, token string) (*authclient.ValidateHECTokenResponse, error)
}

func (m *mockAuthClient) ValidateHECToken(ctx context.Context, token string) (*authclient.ValidateHECTokenResponse, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, token)
	}
	return &authclient.ValidateHECTokenResponse{
		Valid:   true,
		TokenID: "mock-token-id",
		UserID:  "mock-user-id",
	}, nil
}

func TestNewIngestService(t *testing.T) {
	coreClient := &mockNormalizationClient{}
	storageClient := &mockStorageClient{}
	authClient := &mockAuthClient{}

	service := NewIngestService(coreClient, storageClient, authClient)

	if service == nil {
		t.Fatal("NewIngestService() returned nil")
	}

	if service.queueCapacity != 10000 {
		t.Errorf("queueCapacity = %d, want 10000", service.queueCapacity)
	}

	if service.coreClient == nil {
		t.Error("coreClient not set")
	}

	if service.storageClient == nil {
		t.Error("storageClient not set")
	}

	if service.authClient == nil {
		t.Error("authClient not set")
	}

	// Give goroutine time to start before stopping
	time.Sleep(10 * time.Millisecond)
	service.Stop()
	time.Sleep(10 * time.Millisecond) // Let goroutine exit
}

func TestIngestService_IngestEvent(t *testing.T) {
	coreClient := &mockNormalizationClient{}
	storageClient := &mockStorageClient{}
	authClient := &mockAuthClient{}

	service := NewIngestService(coreClient, storageClient, authClient)

	// Give processor goroutine time to start
	time.Sleep(50 * time.Millisecond)

	event := &models.HECEvent{
		Event:      "test event",
		Source:     "test-source",
		SourceType: "test-sourcetype",
		Host:       "test-host",
		Index:      "test-index",
	}

	ackID, err := service.IngestEvent(event, "127.0.0.1", "test-token-id")

	if err != nil {
		t.Fatalf("IngestEvent() error = %v, want nil", err)
	}

	// Without ack manager, ackID should be empty
	if ackID != "" {
		t.Errorf("IngestEvent() ackID = %q, want empty (no ack manager)", ackID)
	}

	// Wait for event to be processed
	time.Sleep(100 * time.Millisecond)

	stats := service.GetStats()
	if stats.TotalEvents != 1 {
		t.Errorf("TotalEvents = %d, want 1", stats.TotalEvents)
	}

	if stats.SuccessfulEvents != 1 {
		t.Errorf("SuccessfulEvents = %d, want 1", stats.SuccessfulEvents)
	}

	service.Stop()
	time.Sleep(10 * time.Millisecond) // Let goroutine exit
}

func TestIngestService_IngestEvent_QueueFull(t *testing.T) {
	// Create service with small queue that blocks on processing
	blockProcessing := make(chan struct{})
	coreClient := &mockNormalizationClient{
		normalizeFunc: func(ctx context.Context, event *models.Event) (*coreclient.NormalizationResult, error) {
			// Block until test releases
			<-blockProcessing
			normalized := map[string]interface{}{"id": event.ID}
			data, _ := json.Marshal(normalized)
			return &coreclient.NormalizationResult{Event: data}, nil
		},
	}
	storageClient := &mockStorageClient{}
	authClient := &mockAuthClient{}

	// Create service with very small capacity
	service := &IngestService{
		eventQueue:    make(chan *models.Event, 2),
		stopChan:      make(chan struct{}),
		queueCapacity: 2,
		stats: models.IngestionStats{
			LastEvent: time.Now(),
		},
		coreClient:    coreClient,
		storageClient: storageClient,
		authClient:    authClient,
	}

	// Start processor
	go service.processEvents()
	defer func() {
		close(blockProcessing) // Unblock processing
		time.Sleep(50 * time.Millisecond)
		service.Stop()
	}()

	event := &models.HECEvent{Event: "test"}

	// Fill the queue (2 events)
	for i := 0; i < 2; i++ {
		_, err := service.IngestEvent(event, "127.0.0.1", "test-token")
		if err != nil {
			t.Fatalf("IngestEvent() %d error = %v", i, err)
		}
	}

	// Wait for processor to pick up first event (leaving 1 in queue)
	time.Sleep(20 * time.Millisecond)

	// Fill queue again
	_, err := service.IngestEvent(event, "127.0.0.1", "test-token")
	if err != nil {
		t.Fatalf("IngestEvent() should succeed, got error: %v", err)
	}

	// This one should fail (queue full: 1 processing + 2 in queue = full)
	_, err = service.IngestEvent(event, "127.0.0.1", "test-token")
	if err == nil {
		t.Error("IngestEvent() with full queue should return error")
		return
	}

	if err.Error() != "event queue full" {
		t.Errorf("IngestEvent() error = %q, want %q", err.Error(), "event queue full")
	}

	stats := service.GetStats()
	if stats.FailedEvents == 0 {
		t.Error("FailedEvents should be > 0 when queue is full")
	}
}

func TestIngestService_IngestRaw(t *testing.T) {
	coreClient := &mockNormalizationClient{}
	storageClient := &mockStorageClient{}
	authClient := &mockAuthClient{}

	service := NewIngestService(coreClient, storageClient, authClient)

	time.Sleep(50 * time.Millisecond)

	rawData := []byte("raw log line")
	ackID, err := service.IngestRaw(rawData, "127.0.0.1", "test-token-id", "test-source", "test-sourcetype", "test-host")

	if err != nil {
		t.Fatalf("IngestRaw() error = %v, want nil", err)
	}

	if ackID != "" {
		t.Errorf("IngestRaw() ackID = %q, want empty (no ack manager)", ackID)
	}

	time.Sleep(100 * time.Millisecond)

	stats := service.GetStats()
	if stats.TotalEvents != 1 {
		t.Errorf("TotalEvents = %d, want 1", stats.TotalEvents)
	}

	if stats.SuccessfulEvents != 1 {
		t.Errorf("SuccessfulEvents = %d, want 1", stats.SuccessfulEvents)
	}

	if stats.TotalBytes != int64(len(rawData)) {
		t.Errorf("TotalBytes = %d, want %d", stats.TotalBytes, len(rawData))
	}

	service.Stop()
	time.Sleep(10 * time.Millisecond)
}

func TestIngestService_ValidateHECToken(t *testing.T) {
	tests := []struct {
		name      string
		mockFunc  func(ctx context.Context, token string) (*authclient.ValidateHECTokenResponse, error)
		token     string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "Valid token",
			mockFunc: func(ctx context.Context, token string) (*authclient.ValidateHECTokenResponse, error) {
				return &authclient.ValidateHECTokenResponse{
					Valid:   true,
					TokenID: "token-123",
					UserID:  "user-456",
				}, nil
			},
			token:   "valid-token",
			wantErr: false,
		},
		{
			name: "Invalid token",
			mockFunc: func(ctx context.Context, token string) (*authclient.ValidateHECTokenResponse, error) {
				return &authclient.ValidateHECTokenResponse{
					Valid: false,
				}, nil
			},
			token:     "invalid-token",
			wantErr:   true,
			errSubstr: "invalid or expired",
		},
		{
			name: "Auth service error",
			mockFunc: func(ctx context.Context, token string) (*authclient.ValidateHECTokenResponse, error) {
				return nil, errors.New("connection timeout")
			},
			token:     "any-token",
			wantErr:   true,
			errSubstr: "token validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authClient := &mockAuthClient{validateFunc: tt.mockFunc}
			service := NewIngestService(&mockNormalizationClient{}, &mockStorageClient{}, authClient)
			time.Sleep(10 * time.Millisecond) // Let goroutine start

			ctx := context.Background()
			err := service.ValidateHECToken(ctx, tt.token)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHECToken() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ValidateHECToken() error = %q, want substring %q", err, tt.errSubstr)
				}
			}

			service.Stop()
			time.Sleep(10 * time.Millisecond) // Let goroutine exit
		})
	}
}

func TestIngestService_ValidateHECToken_NoAuthClient(t *testing.T) {
	// Service without auth client should skip validation
	service := NewIngestService(&mockNormalizationClient{}, &mockStorageClient{}, nil)
	time.Sleep(10 * time.Millisecond)

	ctx := context.Background()
	err := service.ValidateHECToken(ctx, "any-token")

	if err != nil {
		t.Errorf("ValidateHECToken() with nil auth client error = %v, want nil", err)
	}

	service.Stop()
	time.Sleep(10 * time.Millisecond)
}

func TestIngestService_NormalizeEvent_Error(t *testing.T) {
	coreClient := &mockNormalizationClient{
		normalizeFunc: func(ctx context.Context, event *models.Event) (*coreclient.NormalizationResult, error) {
			return nil, errors.New("normalization failed")
		},
	}
	storageClient := &mockStorageClient{}
	authClient := &mockAuthClient{}

	service := NewIngestService(coreClient, storageClient, authClient)

	time.Sleep(50 * time.Millisecond)

	event := &models.HECEvent{Event: "test"}
	_, err := service.IngestEvent(event, "127.0.0.1", "test-token")

	if err != nil {
		t.Fatalf("IngestEvent() error = %v, want nil (queueing should succeed)", err)
	}

	// Wait for processing to fail
	time.Sleep(100 * time.Millisecond)

	// Event was queued but normalization should have failed
	// Check that it doesn't cause service to crash
	stats := service.GetStats()
	if stats.TotalEvents != 1 {
		t.Errorf("TotalEvents = %d, want 1", stats.TotalEvents)
	}

	service.Stop()
	time.Sleep(10 * time.Millisecond)
}

func TestIngestService_StorageError(t *testing.T) {
	coreClient := &mockNormalizationClient{}
	storageClient := &mockStorageClient{
		ingestFunc: func(ctx context.Context, events []map[string]interface{}) (*storageclient.IngestResponse, error) {
			return nil, errors.New("storage unavailable")
		},
	}
	authClient := &mockAuthClient{}

	service := NewIngestService(coreClient, storageClient, authClient)

	time.Sleep(50 * time.Millisecond)

	event := &models.HECEvent{Event: "test"}
	_, err := service.IngestEvent(event, "127.0.0.1", "test-token")

	if err != nil {
		t.Fatalf("IngestEvent() error = %v, want nil (queueing should succeed)", err)
	}

	// Wait for processing to fail
	time.Sleep(100 * time.Millisecond)

	service.Stop()
	time.Sleep(10 * time.Millisecond)
}

func TestIngestService_StoragePartialFailure(t *testing.T) {
	coreClient := &mockNormalizationClient{}
	storageClient := &mockStorageClient{
		ingestFunc: func(ctx context.Context, events []map[string]interface{}) (*storageclient.IngestResponse, error) {
			return &storageclient.IngestResponse{
				Indexed: 0,
				Failed:  1,
				Errors:  []string{"index creation failed"},
			}, nil
		},
	}
	authClient := &mockAuthClient{}

	service := NewIngestService(coreClient, storageClient, authClient)

	time.Sleep(50 * time.Millisecond)

	event := &models.HECEvent{Event: "test"}
	_, err := service.IngestEvent(event, "127.0.0.1", "test-token")

	if err != nil {
		t.Fatalf("IngestEvent() error = %v, want nil (queueing should succeed)", err)
	}

	time.Sleep(100 * time.Millisecond)

	service.Stop()
	time.Sleep(10 * time.Millisecond)
}

func TestIngestService_ParseTime(t *testing.T) {
	service := &IngestService{}

	tests := []struct {
		name     string
		input    *float64
		wantNil  bool
		validate func(t *testing.T, result time.Time)
	}{
		{
			name:    "Nil time - use current time",
			input:   nil,
			wantNil: false,
			validate: func(t *testing.T, result time.Time) {
				now := time.Now()
				diff := now.Sub(result)
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Second {
					t.Errorf("parseTime(nil) too far from now: diff = %v", diff)
				}
			},
		},
		{
			name:    "Epoch time",
			input:   float64Ptr(1234567890.123),
			wantNil: false,
			validate: func(t *testing.T, result time.Time) {
				expected := time.Unix(1234567890, 123000000)
				// Allow small precision difference due to floating point arithmetic
				diff := result.Sub(expected)
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Microsecond {
					t.Errorf("parseTime() = %v, want %v (diff: %v)", result, expected, diff)
				}
			},
		},
		{
			name:    "Whole seconds",
			input:   float64Ptr(1000000000.0),
			wantNil: false,
			validate: func(t *testing.T, result time.Time) {
				expected := time.Unix(1000000000, 0)
				if !result.Equal(expected) {
					t.Errorf("parseTime() = %v, want %v", result, expected)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.parseTime(tt.input)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestIngestService_GetIndex(t *testing.T) {
	service := &IngestService{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty index - use main",
			input:    "",
			expected: "main",
		},
		{
			name:     "Custom index",
			input:    "custom-index",
			expected: "custom-index",
		},
		{
			name:     "Main index explicitly",
			input:    "main",
			expected: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getIndex(tt.input)
			if result != tt.expected {
				t.Errorf("getIndex(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIngestService_GetStats(t *testing.T) {
	service := NewIngestService(&mockNormalizationClient{}, &mockStorageClient{}, &mockAuthClient{})

	time.Sleep(50 * time.Millisecond)

	// Ingest a few events
	for i := 0; i < 3; i++ {
		event := &models.HECEvent{Event: "test"}
		_, err := service.IngestEvent(event, "127.0.0.1", "test-token")
		if err != nil {
			t.Fatalf("IngestEvent() %d error = %v", i, err)
		}
	}

	stats := service.GetStats()

	if stats.TotalEvents != 3 {
		t.Errorf("TotalEvents = %d, want 3", stats.TotalEvents)
	}

	if stats.SuccessfulEvents != 3 {
		t.Errorf("SuccessfulEvents = %d, want 3", stats.SuccessfulEvents)
	}

	if stats.TotalBytes == 0 {
		t.Error("TotalBytes should be > 0")
	}

	if stats.LastEvent.IsZero() {
		t.Error("LastEvent should not be zero")
	}

	service.Stop()
	time.Sleep(10 * time.Millisecond)
}

func TestIngestService_QueryAcks_NoManager(t *testing.T) {
	service := NewIngestService(&mockNormalizationClient{}, &mockStorageClient{}, &mockAuthClient{})
	time.Sleep(10 * time.Millisecond)

	result := service.QueryAcks([]string{"ack-1", "ack-2"})

	if len(result) != 0 {
		t.Errorf("QueryAcks() without ack manager should return empty map, got %d entries", len(result))
	}

	service.Stop()
	time.Sleep(10 * time.Millisecond)
}

func TestIngestService_EventToMap(t *testing.T) {
	service := &IngestService{}

	event := &models.Event{
		ID:         "event-123",
		Timestamp:  time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Host:       "test-host",
		Source:     "test-source",
		SourceType: "test-sourcetype",
		SourceIP:   "127.0.0.1",
		Index:      "test-index",
		Event:      "test event data",
		Fields:     map[string]interface{}{"custom": "field"},
		HECTokenID: "token-456",
		Signature:  "sig-789",
	}

	result := service.eventToMap(event)

	if result["id"] != event.ID {
		t.Errorf("eventToMap() id = %v, want %v", result["id"], event.ID)
	}

	if result["host"] != event.Host {
		t.Errorf("eventToMap() host = %v, want %v", result["host"], event.Host)
	}

	if result["source"] != event.Source {
		t.Errorf("eventToMap() source = %v, want %v", result["source"], event.Source)
	}

	if result["sourcetype"] != event.SourceType {
		t.Errorf("eventToMap() sourcetype = %v, want %v", result["sourcetype"], event.SourceType)
	}

	if result["event"] != event.Event {
		t.Errorf("eventToMap() event = %v, want %v", result["event"], event.Event)
	}
}

func TestIngestService_NoClients(t *testing.T) {
	// Service should work with nil clients (for testing/development)
	service := NewIngestService(nil, nil, nil)

	time.Sleep(50 * time.Millisecond)

	event := &models.HECEvent{Event: "test"}
	_, err := service.IngestEvent(event, "127.0.0.1", "test-token")

	if err != nil {
		t.Fatalf("IngestEvent() error = %v, want nil", err)
	}

	// Should skip normalization and storage but not crash
	time.Sleep(100 * time.Millisecond)

	service.Stop()
	time.Sleep(10 * time.Millisecond)
}

// Helper functions

func float64Ptr(f float64) *float64 {
	return &f
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
