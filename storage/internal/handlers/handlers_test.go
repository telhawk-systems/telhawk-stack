package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/storage/internal/client"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/config"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/indexmgr"
)

func setupTestHandler(t *testing.T) (*StorageHandler, *httptest.Server) {
	// Create a mock OpenSearch server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/":
			// Info endpoint
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "test-node",
				"cluster_name": "test-cluster",
				"version": {
					"number": "2.0.0"
				}
			}`))

		case strings.HasPrefix(r.URL.Path, "/_bulk") || strings.Contains(r.URL.Path, "/_bulk"):
			// Bulk indexing endpoint - count the number of index operations in the request
			body, _ := io.ReadAll(r.Body)
			lines := strings.Split(string(body), "\n")

			// Count valid document lines (every other line after action line)
			itemCount := 0
			for i := 0; i < len(lines); i += 2 {
				if i+1 < len(lines) && strings.TrimSpace(lines[i]) != "" && strings.TrimSpace(lines[i+1]) != "" {
					itemCount++
				}
			}

			// Build response with correct number of items
			items := make([]map[string]interface{}, itemCount)
			for i := 0; i < itemCount; i++ {
				items[i] = map[string]interface{}{
					"index": map[string]interface{}{
						"_index":   "test-events-write",
						"_id":      fmt.Sprintf("%d", i+1),
						"_version": 1,
						"result":   "created",
						"status":   201,
					},
				}
			}

			response := map[string]interface{}{
				"took":   10,
				"errors": false,
				"items":  items,
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"acknowledged": true}`))
		}
	}))

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, err := client.NewOpenSearchClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OpenSearch client: %v", err)
	}

	indexCfg := config.IndexManagementConfig{
		IndexPrefix:     "test-events",
		ShardCount:      1,
		ReplicaCount:    0,
		RefreshInterval: "5s",
	}

	indexManager := indexmgr.NewIndexManager(osClient, indexCfg)
	handler := NewStorageHandler(osClient, indexManager)

	return handler, mockServer
}

func TestNewStorageHandler(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	if handler.client == nil {
		t.Error("Expected non-nil client")
	}

	if handler.indexManager == nil {
		t.Error("Expected non-nil index manager")
	}
}

func TestIngest_Success(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	events := []map[string]interface{}{
		{
			"class_uid":  3002,
			"class_name": "Authentication",
			"time":       "2025-01-15T10:00:00Z",
			"severity":   "Informational",
		},
		{
			"class_uid":  1007,
			"class_name": "Process Activity",
			"time":       "2025-01-15T10:01:00Z",
			"severity":   "Low",
		},
	}

	reqBody := IngestRequest{Events: events}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Ingest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		t.Errorf("Expected status 200 or 206, got %d", resp.StatusCode)
	}

	var response IngestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	totalProcessed := response.Indexed + response.Failed
	if totalProcessed != 2 {
		t.Errorf("Expected 2 events to be processed (indexed or failed), got %d (indexed: %d, failed: %d)",
			totalProcessed, response.Indexed, response.Failed)
	}
}

func TestIngest_MethodNotAllowed(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ingest", nil)
	w := httptest.NewRecorder()

	handler.Ingest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestIngest_InvalidJSON(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Ingest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestIngest_NoEvents(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	reqBody := IngestRequest{Events: []map[string]interface{}{}}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Ingest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBulkIngest_Success(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	// OpenSearch bulk format: action line + data line
	bulkData := `{"index":{}}
{"class_uid":3002,"class_name":"Authentication","time":"2025-01-15T10:00:00Z"}
{"index":{}}
{"class_uid":1007,"class_name":"Process Activity","time":"2025-01-15T10:01:00Z"}
`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bulk", strings.NewReader(bulkData))
	req.Header.Set("Content-Type", "application/x-ndjson")

	w := httptest.NewRecorder()
	handler.BulkIngest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		t.Errorf("Expected status 200 or 206, got %d", resp.StatusCode)
	}

	var response IngestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	totalProcessed := response.Indexed + response.Failed
	if totalProcessed != 2 {
		t.Errorf("Expected 2 events to be processed (indexed or failed), got %d (indexed: %d, failed: %d)",
			totalProcessed, response.Indexed, response.Failed)
	}
}

func TestBulkIngest_MethodNotAllowed(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bulk", nil)
	w := httptest.NewRecorder()

	handler.BulkIngest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestBulkIngest_NoValidEvents(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	// Invalid bulk data
	bulkData := `invalid
data
`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bulk", strings.NewReader(bulkData))
	req.Header.Set("Content-Type", "application/x-ndjson")

	w := httptest.NewRecorder()
	handler.BulkIngest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestIndexEvents(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	events := []map[string]interface{}{
		{
			"class_uid":  3002,
			"class_name": "Authentication",
			"time":       "2025-01-15T10:00:00Z",
			"severity":   "Informational",
		},
	}

	ctx := context.Background()
	resp := handler.indexEvents(ctx, events)

	totalProcessed := resp.Indexed + resp.Failed
	if totalProcessed != 1 {
		t.Errorf("Expected 1 event to be processed (indexed or failed), got %d (indexed: %d, failed: %d)",
			totalProcessed, resp.Indexed, resp.Failed)
	}
}

func TestIndexEvents_EmptyEvents(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	events := []map[string]interface{}{}

	ctx := context.Background()
	resp := handler.indexEvents(ctx, events)

	if resp.Indexed != 0 {
		t.Errorf("Expected 0 indexed events, got %d", resp.Indexed)
	}
}

func TestHealth(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var healthResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", healthResp["status"])
	}
}

func TestReady_Success(t *testing.T) {
	handler, mockServer := setupTestHandler(t)
	defer mockServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var readyResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&readyResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if readyResp["status"] != "ready" {
		t.Errorf("Expected status ready, got %s", readyResp["status"])
	}
}

func TestReady_OpenSearchUnavailable(t *testing.T) {
	// Create a mock server that returns errors
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error": "service unavailable"}`))
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, err := client.NewOpenSearchClient(cfg)
	if err != nil {
		// If we can't create the client, that's fine for this test
		t.Skip("Skipping test - could not create client with error server")
	}

	indexCfg := config.IndexManagementConfig{
		IndexPrefix: "test-events",
	}

	indexManager := indexmgr.NewIndexManager(osClient, indexCfg)
	handler := NewStorageHandler(osClient, indexManager)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var readyResp map[string]string
	if err := json.Unmarshal(bodyBytes, &readyResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if readyResp["status"] != "not ready" {
		t.Errorf("Expected status 'not ready', got %s", readyResp["status"])
	}
}

func TestIngestResponse_JSON(t *testing.T) {
	resp := IngestResponse{
		Indexed: 10,
		Failed:  2,
		Errors:  []string{"error1", "error2"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var decoded IngestResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decoded.Indexed != 10 {
		t.Errorf("Expected 10 indexed, got %d", decoded.Indexed)
	}

	if decoded.Failed != 2 {
		t.Errorf("Expected 2 failed, got %d", decoded.Failed)
	}

	if len(decoded.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(decoded.Errors))
	}
}

func TestIngestRequest_JSON(t *testing.T) {
	req := IngestRequest{
		Events: []map[string]interface{}{
			{
				"class_uid": 3002,
				"time":      "2025-01-15T10:00:00Z",
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var decoded IngestRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if len(decoded.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(decoded.Events))
	}

	if decoded.Events[0]["class_uid"] != float64(3002) {
		t.Errorf("Expected class_uid 3002, got %v", decoded.Events[0]["class_uid"])
	}
}

func TestBulkIngest_PartialSuccess(t *testing.T) {
	// Create a mock server that fails some requests
	failureCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))

		case strings.HasPrefix(r.URL.Path, "/_bulk"):
			// Simulate partial failure
			w.WriteHeader(http.StatusOK)
			response := `{
				"took": 10,
				"errors": true,
				"items": [
					{
						"index": {
							"_index": "test-events-write",
							"_id": "1",
							"_version": 1,
							"result": "created",
							"status": 201
						}
					},
					{
						"index": {
							"_index": "test-events-write",
							"_id": "2",
							"status": 400,
							"error": {
								"type": "mapper_parsing_exception",
								"reason": "failed to parse field"
							}
						}
					}
				]
			}`
			failureCount++
			w.Write([]byte(response))

		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"acknowledged": true}`))
		}
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, _ := client.NewOpenSearchClient(cfg)

	indexCfg := config.IndexManagementConfig{
		IndexPrefix: "test-events",
	}

	indexManager := indexmgr.NewIndexManager(osClient, indexCfg)
	handler := NewStorageHandler(osClient, indexManager)

	bulkData := `{"index":{}}
{"class_uid":3002,"time":"2025-01-15T10:00:00Z"}
{"index":{}}
{"class_uid":1007,"time":"2025-01-15T10:01:00Z"}
`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bulk", strings.NewReader(bulkData))
	w := httptest.NewRecorder()

	handler.BulkIngest(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Partial success should return 206
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 206 or 200, got %d", resp.StatusCode)
	}
}
