package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueryClient(t *testing.T) {
	client := NewQueryClient("http://localhost:3000")

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:3000", client.baseURL)
	assert.NotNil(t, client.client)
	assert.Equal(t, 30*time.Second, client.client.Timeout)
}

func TestQueryClient_Client(t *testing.T) {
	client := NewQueryClient("http://localhost:3000")

	httpClient := client.Client()
	assert.NotNil(t, httpClient)
	assert.Equal(t, client.client, httpClient)
}

// writeJSONAPIResponse is a helper to send JSON:API formatted responses
func writeJSONAPIResponse(w http.ResponseWriter, results []map[string]interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusOK)
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "search-result",
			"id":   "test-request-id",
			"attributes": map[string]interface{}{
				"request_id":    "test-request-id",
				"latency_ms":    10,
				"result_count":  len(results),
				"total_matches": len(results),
				"results":       results,
			},
		},
	}
	json.NewEncoder(w).Encode(resp)
}

// writeJSONAPIError is a helper to send JSON:API error responses
func writeJSONAPIErrorResponse(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	resp := map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"status": http.StatusText(status),
				"code":   code,
				"title":  code,
				"detail": detail,
			},
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func TestSearch_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/query/v1/search", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Contains(t, r.Header.Get("Content-Type"), "application/vnd.api+json")
		assert.Contains(t, r.Header.Get("Accept"), "application/vnd.api+json")

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		// Verify JSON:API structure
		data, ok := payload["data"].(map[string]interface{})
		require.True(t, ok, "payload should have data field")
		assert.Equal(t, "search", data["type"])

		attrs, ok := data["attributes"].(map[string]interface{})
		require.True(t, ok, "data should have attributes field")
		assert.Equal(t, "severity:high", attrs["query"])
		assert.NotNil(t, attrs["time_range"])

		writeJSONAPIResponse(w, []map[string]interface{}{
			{
				"time":     1700000000,
				"severity": "high",
				"message":  "Failed login attempt",
				"user":     "admin",
			},
			{
				"time":     1700000100,
				"severity": "high",
				"message":  "Port scan detected",
				"src_ip":   "192.168.1.100",
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "severity:high", "-1h", "now", "")

	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify first result
	assert.Equal(t, "high", results[0]["severity"])
	assert.Equal(t, "Failed login attempt", results[0]["message"])
	assert.Equal(t, "admin", results[0]["user"])

	// Verify second result
	assert.Equal(t, "high", results[1]["severity"])
	assert.Equal(t, "Port scan detected", results[1]["message"])
	assert.Equal(t, "192.168.1.100", results[1]["src_ip"])
}

func TestSearch_EmptyResults(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONAPIResponse(w, []map[string]interface{}{})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "nonexistent:query", "-1h", "now", "")

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearch_WithLastParameter(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		data := payload["data"].(map[string]interface{})
		attrs := data["attributes"].(map[string]interface{})

		// When "last" is provided, time_range should be computed from it
		timeRange, ok := attrs["time_range"].(map[string]interface{})
		require.True(t, ok, "should have time_range")
		assert.NotNil(t, timeRange["from"])
		assert.NotNil(t, timeRange["to"])

		writeJSONAPIResponse(w, []map[string]interface{}{})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "*", "", "", "24h")

	require.NoError(t, err)
	assert.NotNil(t, results)
}

func TestSearch_ComplexQuery(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		data := payload["data"].(map[string]interface{})
		attrs := data["attributes"].(map[string]interface{})
		assert.Equal(t, "class_uid:3002 AND status_id:2 AND actor.user.name:admin", attrs["query"])

		writeJSONAPIResponse(w, []map[string]interface{}{
			{
				"class_uid":  3002,
				"status_id":  2,
				"class_name": "Authentication",
				"actor": map[string]interface{}{
					"user": map[string]interface{}{
						"name": "admin",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "class_uid:3002 AND status_id:2 AND actor.user.name:admin", "-1d", "now", "")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, float64(3002), results[0]["class_uid"]) // JSON numbers are float64
}

func TestSearch_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONAPIErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Invalid token")
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search("invalid-token", "test", "-1h", "now", "")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestSearch_BadRequest(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONAPIErrorResponse(w, http.StatusBadRequest, "invalid_query", "Invalid query syntax")
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "invalid:syntax:query:", "-1h", "now", "")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "invalid_query")
}

func TestSearch_ServerError(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONAPIErrorResponse(w, http.StatusInternalServerError, "opensearch_error", "OpenSearch unavailable")
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "*", "-1h", "now", "")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "opensearch_error")
}

func TestSearch_NetworkError(t *testing.T) {
	client := NewQueryClient("http://invalid-host-does-not-exist.local:99999")
	results, err := client.Search("token", "test", "-1h", "now", "")

	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestSearch_InvalidJSON(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json response`))
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "test", "-1h", "now", "")

	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestSearch_TimeRanges(t *testing.T) {
	tests := []struct {
		name     string
		earliest string
		latest   string
		last     string
	}{
		{
			name:     "relative time with last hour",
			earliest: "-1h",
			latest:   "now",
			last:     "",
		},
		{
			name:     "last 24 hours shorthand",
			earliest: "",
			latest:   "",
			last:     "24h",
		},
		{
			name:     "last 7 days shorthand",
			earliest: "",
			latest:   "",
			last:     "7d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testToken := createTestJWT("user-123")

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]interface{}
				json.NewDecoder(r.Body).Decode(&payload)

				data := payload["data"].(map[string]interface{})
				attrs := data["attributes"].(map[string]interface{})

				// All cases should have a time_range computed
				timeRange, ok := attrs["time_range"].(map[string]interface{})
				require.True(t, ok, "should have time_range")
				assert.NotNil(t, timeRange["from"])
				assert.NotNil(t, timeRange["to"])

				writeJSONAPIResponse(w, []map[string]interface{}{})
			}))
			defer server.Close()

			client := NewQueryClient(server.URL)
			_, err := client.Search(testToken, "*", tt.earliest, tt.latest, tt.last)
			assert.NoError(t, err)
		})
	}
}

func TestSearch_LargeResultSet(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate 1000 results
		results := make([]map[string]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			results[i] = map[string]interface{}{
				"id":      i,
				"message": "Event " + string(rune(i)),
			}
		}

		writeJSONAPIResponse(w, results)
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "*", "-1d", "now", "")

	require.NoError(t, err)
	assert.Len(t, results, 1000)
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"1h", time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}

func TestParseTimeSpec(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		input    string
		hasError bool
	}{
		{"now", false},
		{"-1h", false},
		{"-7d", false},
		{"2024-01-15T10:30:00Z", false},
		{"2024-01-15", false},
		{"invalid-time", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseTimeSpec(tt.input, now)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseTimeRange(t *testing.T) {
	// Test with "last" parameter
	tr, err := parseTimeRange("", "", "1h")
	assert.NoError(t, err)
	assert.NotNil(t, tr)
	assert.True(t, tr.To.After(tr.From))

	// Test with earliest/latest
	tr, err = parseTimeRange("-1h", "now", "")
	assert.NoError(t, err)
	assert.NotNil(t, tr)
	assert.True(t, tr.To.After(tr.From))

	// Test default (no params - defaults to last 24h)
	tr, err = parseTimeRange("", "", "")
	assert.NoError(t, err)
	assert.NotNil(t, tr)
	assert.True(t, tr.To.After(tr.From))

	// Test invalid duration
	_, err = parseTimeRange("", "", "invalid")
	assert.Error(t, err)
}
