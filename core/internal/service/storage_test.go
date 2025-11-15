package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer/generated"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/service"
	"github.com/telhawk-systems/telhawk-stack/core/internal/storage"
	"github.com/telhawk-systems/telhawk-stack/core/internal/validator"
)

// TestStoragePersistence verifies that normalized events are stored successfully
func TestStoragePersistence(t *testing.T) {
	// Track storage calls
	var storedEvents []map[string]interface{}
	var storageCallCount int

	// Mock storage server
	storageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storageCallCount++

		if r.URL.Path != "/api/v1/ingest" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req struct {
			Events []map[string]interface{} `json:"events"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		storedEvents = append(storedEvents, req.Events...)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"indexed": len(req.Events),
			"failed":  0,
		})
	}))
	defer storageServer.Close()

	// Setup pipeline with generated normalizers
	registry := normalizer.NewRegistry(
		generated.NewAuthenticationNormalizer(),
		generated.NewNetworkActivityNormalizer(),
	)
	validators := validator.NewChain(validator.BasicValidator{})
	pipe := pipeline.New(registry, validators)

	// Setup storage client and processor
	storageClient := storage.NewClient(storageServer.URL)
	processor := service.NewProcessor(pipe, storageClient, nil) // nil DLQ for tests

	testCases := []struct {
		name        string
		sourceType  string
		payload     string
		expectStore bool
		expectClass string
	}{
		{
			name:        "Authentication Event",
			sourceType:  "auth_login",
			payload:     `{"user":"alice","action":"login","status":"success","timestamp":"2025-11-03T23:44:00Z"}`,
			expectStore: true,
			expectClass: "authentication",
		},
		{
			name:        "Network Event",
			sourceType:  "network_firewall",
			payload:     `{"src_ip":"10.0.0.1","dst_ip":"8.8.8.8","dst_port":443,"protocol":"tcp","action":"allow","timestamp":"2025-11-03T23:44:00Z"}`,
			expectStore: true,
			expectClass: "network_activity",
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			beforeCount := len(storedEvents)

			envelope := &model.RawEventEnvelope{
				Format:     "json",
				SourceType: tc.sourceType,
				Source:     "test",
				Payload:    []byte(tc.payload),
				ReceivedAt: time.Now(),
			}

			event, err := processor.Process(ctx, envelope)
			if err != nil {
				t.Fatalf("processor failed: %v", err)
			}

			if event == nil {
				t.Fatal("expected non-nil event")
			}

			if event.Class != tc.expectClass {
				t.Errorf("expected class %q, got %q", tc.expectClass, event.Class)
			}

			if tc.expectStore {
				afterCount := len(storedEvents)
				if afterCount != beforeCount+1 {
					t.Errorf("expected 1 event stored, got %d", afterCount-beforeCount)
				}

				if afterCount > 0 {
					stored := storedEvents[afterCount-1]
					if stored["class"] != tc.expectClass {
						t.Errorf("stored event has wrong class: %v", stored["class"])
					}
					t.Logf("✓ Event stored successfully: class=%s, class_uid=%v",
						stored["class"], stored["class_uid"])
				}
			}
		})
	}

	// Verify health stats
	stats := processor.Health()
	if stats.Processed != 2 {
		t.Errorf("expected 2 processed events, got %d", stats.Processed)
	}
	if stats.Stored != 2 {
		t.Errorf("expected 2 stored events, got %d", stats.Stored)
	}
	if stats.Failed != 0 {
		t.Errorf("expected 0 failed events, got %d", stats.Failed)
	}

	t.Logf("Final stats: processed=%d, stored=%d, failed=%d, calls=%d",
		stats.Processed, stats.Stored, stats.Failed, storageCallCount)
}

// TestStorageRetry verifies retry logic on temporary failures
func TestStorageRetry(t *testing.T) {
	callCount := 0
	failUntil := 2

	storageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < failUntil {
			// Simulate temporary server error
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"temporarily unavailable"}`))
			return
		}

		// Success on retry
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"indexed": 1,
			"failed":  0,
		})
	}))
	defer storageServer.Close()

	registry := normalizer.NewRegistry(generated.NewAuthenticationNormalizer())
	validators := validator.NewChain(validator.BasicValidator{})
	pipe := pipeline.New(registry, validators)
	storageClient := storage.NewClient(storageServer.URL)
	processor := service.NewProcessor(pipe, storageClient, nil)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "auth_login",
		Source:     "test",
		Payload:    []byte(`{"user":"bob","action":"login","status":"success"}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := processor.Process(ctx, envelope)

	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}

	if callCount < failUntil {
		t.Errorf("expected at least %d calls (with retries), got %d", failUntil, callCount)
	}

	t.Logf("✓ Storage succeeded after %d attempts (including %d failures)", callCount, failUntil-1)
}

// TestStorageFailure verifies error handling when storage is unavailable
func TestStorageFailure(t *testing.T) {
	// Create server that always fails
	storageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"storage unavailable"}`))
	}))
	defer storageServer.Close()

	registry := normalizer.NewRegistry(generated.NewAuthenticationNormalizer())
	validators := validator.NewChain(validator.BasicValidator{})
	pipe := pipeline.New(registry, validators)
	storageClient := storage.NewClient(storageServer.URL)
	processor := service.NewProcessor(pipe, storageClient, nil)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "auth_login",
		Source:     "test",
		Payload:    []byte(`{"user":"charlie","action":"login","status":"success"}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	_, err := processor.Process(ctx, envelope)

	if err == nil {
		t.Fatal("expected error when storage fails, got nil")
	}

	stats := processor.Health()
	if stats.Failed == 0 {
		t.Error("expected failed counter to increment")
	}

	t.Logf("✓ Error properly returned: %v", err)
	t.Logf("✓ Failed counter: %d", stats.Failed)
}
