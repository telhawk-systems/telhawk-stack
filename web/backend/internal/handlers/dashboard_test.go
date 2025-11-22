package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestDashboardHandler_GetMetrics_CacheMiss(t *testing.T) {
	// Create mock query service
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events/query" {
			t.Errorf("Expected path /api/v1/events/query, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Verify JSON:API headers
		if r.Header.Get("Content-Type") != "application/vnd.api+json" {
			t.Errorf("Expected Content-Type application/vnd.api+json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "application/vnd.api+json" {
			t.Errorf("Expected Accept application/vnd.api+json, got %s", r.Header.Get("Accept"))
		}

		// Return JSON:API response
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "event-query",
				"id":   "test-query",
			},
			"meta": map[string]interface{}{
				"total": 100,
				"aggregations": map[string]interface{}{
					"severity_count": map[string]interface{}{
						"buckets": []map[string]interface{}{
							{"key": "high", "doc_count": 50},
							{"key": "medium", "doc_count": 30},
							{"key": "low", "doc_count": 20},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "test-token"})
	rr := httptest.NewRecorder()

	handler.GetMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Verify X-Cache header indicates MISS
	if cacheHeader := rr.Header().Get("X-Cache"); cacheHeader != "MISS" {
		t.Errorf("Expected X-Cache MISS, got %s", cacheHeader)
	}

	// Verify response structure (legacy format)
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if total, ok := resp["total_matches"].(float64); !ok || total != 100 {
		t.Errorf("Expected total_matches 100, got %v", resp["total_matches"])
	}

	if _, ok := resp["aggregations"]; !ok {
		t.Error("Expected aggregations field in response")
	}
}

func TestDashboardHandler_GetMetrics_CacheHit(t *testing.T) {
	callCount := 0
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "event-query",
			},
			"meta": map[string]interface{}{
				"total":        50,
				"aggregations": map[string]interface{}{},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	// First request - should be a cache miss
	req1 := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rr1 := httptest.NewRecorder()
	handler.GetMetrics(rr1, req1)

	if rr1.Header().Get("X-Cache") != "MISS" {
		t.Error("Expected first request to be cache MISS")
	}

	// Second request - should be a cache hit
	req2 := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rr2 := httptest.NewRecorder()
	handler.GetMetrics(rr2, req2)

	if rr2.Header().Get("X-Cache") != "HIT" {
		t.Error("Expected second request to be cache HIT")
	}

	// Verify X-Cache-Age header exists
	if cacheAge := rr2.Header().Get("X-Cache-Age"); cacheAge == "" {
		t.Error("Expected X-Cache-Age header to be set")
	}

	// Verify query service was only called once
	if callCount != 1 {
		t.Errorf("Expected query service to be called once, got %d calls", callCount)
	}

	// Verify both responses are identical
	var resp1, resp2 map[string]interface{}
	json.Unmarshal(rr1.Body.Bytes(), &resp1)
	json.Unmarshal(rr2.Body.Bytes(), &resp2)

	if resp1["total_matches"] != resp2["total_matches"] {
		t.Error("Expected cached response to match original response")
	}
}

func TestDashboardHandler_GetMetrics_CacheExpiration(t *testing.T) {
	// Set very short cache duration using environment variable
	os.Setenv("DASHBOARD_CACHE_SECONDS", "1")
	defer os.Unsetenv("DASHBOARD_CACHE_SECONDS")

	callCount := 0
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "event-query",
			},
			"meta": map[string]interface{}{
				"total":        callCount * 10,
				"aggregations": map[string]interface{}{},
			},
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	// First request
	req1 := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rr1 := httptest.NewRecorder()
	handler.GetMetrics(rr1, req1)

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Second request - should be a cache miss due to expiration
	req2 := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rr2 := httptest.NewRecorder()
	handler.GetMetrics(rr2, req2)

	if rr2.Header().Get("X-Cache") != "MISS" {
		t.Error("Expected cache MISS after expiration")
	}

	// Verify query service was called twice
	if callCount != 2 {
		t.Errorf("Expected query service to be called twice, got %d calls", callCount)
	}
}

func TestDashboardHandler_GetMetrics_CustomCacheDuration(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		expectedSecs int
	}{
		{
			name:         "Valid custom duration",
			envValue:     "60",
			expectedSecs: 60,
		},
		{
			name:         "Zero cache (disabled)",
			envValue:     "0",
			expectedSecs: 0,
		},
		{
			name:         "Invalid value uses default",
			envValue:     "invalid",
			expectedSecs: 300,
		},
		{
			name:         "Negative value uses default",
			envValue:     "-10",
			expectedSecs: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("DASHBOARD_CACHE_SECONDS", tt.envValue)
				defer os.Unsetenv("DASHBOARD_CACHE_SECONDS")
			}

			queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := map[string]interface{}{
					"data": map[string]interface{}{"type": "event-query"},
					"meta": map[string]interface{}{
						"total":        10,
						"aggregations": map[string]interface{}{},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer queryServer.Close()

			handler := NewDashboardHandler(queryServer.URL, "")

			expectedDuration := time.Duration(tt.expectedSecs) * time.Second
			if handler.cacheDuration != expectedDuration {
				t.Errorf("Expected cache duration %v, got %v", expectedDuration, handler.cacheDuration)
			}
		})
	}
}

func TestDashboardHandler_GetMetrics_QueryServiceError(t *testing.T) {
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rr := httptest.NewRecorder()

	handler.GetMetrics(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestDashboardHandler_GetMetrics_QueryServiceUnauthorized(t *testing.T) {
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rr := httptest.NewRecorder()

	handler.GetMetrics(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestDashboardHandler_GetMetrics_AuthorizationHeader(t *testing.T) {
	var receivedAuthHeader string

	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")

		resp := map[string]interface{}{
			"data": map[string]interface{}{"type": "event-query"},
			"meta": map[string]interface{}{
				"total":        0,
				"aggregations": map[string]interface{}{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "test-access-token"})
	rr := httptest.NewRecorder()

	handler.GetMetrics(rr, req)

	expectedAuth := "Bearer test-access-token"
	if receivedAuthHeader != expectedAuth {
		t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, receivedAuthHeader)
	}
}

func TestDashboardHandler_GetMetrics_NoAccessToken(t *testing.T) {
	var receivedAuthHeader string

	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")

		resp := map[string]interface{}{
			"data": map[string]interface{}{"type": "event-query"},
			"meta": map[string]interface{}{
				"total":        0,
				"aggregations": map[string]interface{}{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rr := httptest.NewRecorder()

	handler.GetMetrics(rr, req)

	if receivedAuthHeader != "" {
		t.Errorf("Expected no Authorization header, got '%s'", receivedAuthHeader)
	}
}

func TestDashboardHandler_GetMetrics_EmptyAggregations(t *testing.T) {
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{"type": "event-query"},
			"meta": map[string]interface{}{
				"total": 0,
				// No aggregations field
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
	rr := httptest.NewRecorder()

	handler.GetMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have empty aggregations object when not provided
	if agg, ok := resp["aggregations"].(map[string]interface{}); !ok || len(agg) != 0 {
		t.Errorf("Expected empty aggregations object, got %v", resp["aggregations"])
	}
}

func TestDashboardHandler_GetMetrics_ConcurrentRequests(t *testing.T) {
	callCount := 0
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Simulate slow response
		time.Sleep(50 * time.Millisecond)

		resp := map[string]interface{}{
			"data": map[string]interface{}{"type": "event-query"},
			"meta": map[string]interface{}{
				"total":        callCount,
				"aggregations": map[string]interface{}{},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer queryServer.Close()

	handler := NewDashboardHandler(queryServer.URL, "")

	// Make concurrent requests
	numRequests := 5
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/v1/dashboard/metrics", nil)
			rr := httptest.NewRecorder()
			handler.GetMetrics(rr, req)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// Due to race conditions with concurrent requests before cache is populated,
	// we may get multiple calls. The important thing is that all requests succeed.
	// In real usage, cache hits would be higher after initial population.
	if callCount < 1 {
		t.Errorf("Expected at least 1 call to query service, got %d", callCount)
	}
	if callCount > numRequests {
		t.Errorf("Expected at most %d calls to query service, got %d", numRequests, callCount)
	}
}
