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

func TestNewIngestClient(t *testing.T) {
	client := NewIngestClient("http://localhost:8088")

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8088", client.baseURL)
	assert.NotNil(t, client.client)
	assert.Equal(t, 10*time.Second, client.client.Timeout)
}

func TestSendEvent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/services/collector/event", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Splunk test-hec-token-123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		// Verify payload structure
		assert.Contains(t, payload, "event")
		assert.Contains(t, payload, "source")
		assert.Contains(t, payload, "sourcetype")
		assert.Contains(t, payload, "time")

		// Verify event data
		event := payload["event"].(map[string]interface{})
		assert.Equal(t, "test message", event["message"])
		assert.Equal(t, "error", event["severity"])

		// Verify metadata
		assert.Equal(t, "cli-test", payload["source"])
		assert.Equal(t, "json", payload["sourcetype"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text":"Success","code":0}`))
	}))
	defer server.Close()

	client := NewIngestClient(server.URL)
	event := map[string]interface{}{
		"message":  "test message",
		"severity": "error",
	}

	err := client.SendEvent("test-hec-token-123", event, "cli-test", "json")
	assert.NoError(t, err)
}

func TestSendEvent_ComplexEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		event := payload["event"].(map[string]interface{})
		assert.Equal(t, "admin", event["user"])
		assert.Equal(t, "192.168.1.100", event["source_ip"])
		assert.Equal(t, float64(22), event["port"]) // JSON numbers are float64

		// Verify nested object
		metadata := event["metadata"].(map[string]interface{})
		assert.Equal(t, "authentication", metadata["type"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewIngestClient(server.URL)
	event := map[string]interface{}{
		"user":      "admin",
		"source_ip": "192.168.1.100",
		"port":      22,
		"metadata": map[string]interface{}{
			"type": "authentication",
		},
	}

	err := client.SendEvent("token", event, "test", "ocsf")
	assert.NoError(t, err)
}

func TestSendEvent_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"text":"Invalid token","code":4}`))
	}))
	defer server.Close()

	client := NewIngestClient(server.URL)
	event := map[string]interface{}{"test": "data"}

	err := client.SendEvent("invalid-token", event, "test", "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ingest failed with status 401")
}

func TestSendEvent_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"text":"Internal error","code":8}`))
	}))
	defer server.Close()

	client := NewIngestClient(server.URL)
	event := map[string]interface{}{"test": "data"}

	err := client.SendEvent("token", event, "test", "json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ingest failed with status 500")
}

func TestSendEvent_NetworkError(t *testing.T) {
	client := NewIngestClient("http://invalid-host-does-not-exist.local:99999")
	event := map[string]interface{}{"test": "data"}

	err := client.SendEvent("token", event, "test", "json")
	assert.Error(t, err)
}

func TestSendEvent_TimestampAdded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		// Verify timestamp is present and recent
		assert.Contains(t, payload, "time")
		timestamp := int64(payload["time"].(float64))
		now := time.Now().Unix()
		assert.InDelta(t, now, timestamp, 2) // Within 2 seconds

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewIngestClient(server.URL)
	event := map[string]interface{}{"message": "test"}

	err := client.SendEvent("token", event, "test", "json")
	assert.NoError(t, err)
}

func TestSendEvent_EmptyEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		event := payload["event"].(map[string]interface{})
		assert.Empty(t, event) // Empty event map is valid

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewIngestClient(server.URL)
	event := map[string]interface{}{} // Empty event

	err := client.SendEvent("token", event, "test", "json")
	assert.NoError(t, err)
}

func TestSendEvent_SourceAndSourcetype(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		sourcetype string
	}{
		{
			name:       "standard values",
			source:     "app-server-01",
			sourcetype: "syslog",
		},
		{
			name:       "empty source",
			source:     "",
			sourcetype: "json",
		},
		{
			name:       "empty sourcetype",
			source:     "test",
			sourcetype: "",
		},
		{
			name:       "both empty",
			source:     "",
			sourcetype: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]interface{}
				json.NewDecoder(r.Body).Decode(&payload)

				assert.Equal(t, tt.source, payload["source"])
				assert.Equal(t, tt.sourcetype, payload["sourcetype"])

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewIngestClient(server.URL)
			event := map[string]interface{}{"test": "data"}

			err := client.SendEvent("token", event, tt.source, tt.sourcetype)
			assert.NoError(t, err)
		})
	}
}
