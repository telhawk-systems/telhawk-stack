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
	client := NewQueryClient("http://localhost:8082")

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8082", client.baseURL)
	assert.NotNil(t, client.client)
	assert.Equal(t, 30*time.Second, client.client.Timeout)
}

func TestQueryClient_Client(t *testing.T) {
	client := NewQueryClient("http://localhost:8082")

	httpClient := client.Client()
	assert.NotNil(t, httpClient)
	assert.Equal(t, client.client, httpClient)
}

func TestSearch_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/search", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		assert.Equal(t, "severity:high", payload["query"])
		assert.Equal(t, "-1h", payload["earliest"])
		assert.Equal(t, "now", payload["latest"])
		assert.Equal(t, "", payload["last"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{})
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

		assert.Equal(t, "24h", payload["last"])
		assert.Equal(t, "", payload["earliest"]) // last takes precedence
		assert.Equal(t, "", payload["latest"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{})
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

		assert.Equal(t, "class_uid:3002 AND status_id:2 AND actor.user.name:admin", payload["query"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
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
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search("invalid-token", "test", "-1h", "now", "")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "search failed with status 401")
}

func TestSearch_BadRequest(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid query syntax"}`))
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "invalid:syntax:query:", "-1h", "now", "")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "search failed with status 400")
}

func TestSearch_ServerError(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"opensearch unavailable"}`))
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "*", "-1h", "now", "")

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "search failed with status 500")
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
			name:     "last 24 hours",
			earliest: "",
			latest:   "",
			last:     "24h",
		},
		{
			name:     "absolute timestamps",
			earliest: "1700000000",
			latest:   "1700100000",
			last:     "",
		},
		{
			name:     "last 7 days",
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

				assert.Equal(t, tt.earliest, payload["earliest"])
				assert.Equal(t, tt.latest, payload["latest"])
				assert.Equal(t, tt.last, payload["last"])

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]interface{}{})
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

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(results)
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	results, err := client.Search(testToken, "*", "-1d", "now", "")

	require.NoError(t, err)
	assert.Len(t, results, 1000)
}
