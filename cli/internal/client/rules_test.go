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

func TestNewRulesClient(t *testing.T) {
	client := NewRulesClient("http://localhost:8084")

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8084", client.baseURL)
	assert.NotNil(t, client.client)
	assert.Equal(t, 30*time.Second, client.client.Timeout)
}

func TestListSchemas_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/schemas", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "1", r.URL.Query().Get("page[number]"))
		assert.Equal(t, "10", r.URL.Query().Get("page[size]"))
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.api+json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/vnd.api+json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []DetectionSchema{
				{
					Type: "detection-schema",
					ID:   "schema-1",
					Attributes: DetectionSchemaAttributes{
						VersionID:  "v1",
						Model:      map[string]interface{}{"correlation": "event_count"},
						View:       map[string]interface{}{"name": "Failed Logins"},
						Controller: map[string]interface{}{"enabled": true},
						CreatedAt:  time.Now(),
					},
				},
			},
			"meta": map[string]interface{}{
				"total": 1,
				"page":  1,
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	schemas, meta, err := client.ListSchemas(testToken, 1, 10)

	require.NoError(t, err)
	assert.Len(t, schemas, 1)
	assert.Equal(t, "schema-1", schemas[0].ID)
	assert.Equal(t, "Failed Logins", schemas[0].Attributes.View["name"])
	assert.NotNil(t, meta)
	assert.Equal(t, float64(1), meta["total"])
}

func TestListSchemas_Pagination(t *testing.T) {
	testToken := createTestJWT("user-123")

	tests := []struct {
		name  string
		page  int
		limit int
	}{
		{"first page", 1, 20},
		{"second page", 2, 20},
		{"large page size", 1, 100},
		{"small page size", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.page, mustAtoi(r.URL.Query().Get("page[number]")))
				assert.Equal(t, tt.limit, mustAtoi(r.URL.Query().Get("page[size]")))

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": []DetectionSchema{},
					"meta": map[string]interface{}{},
				})
			}))
			defer server.Close()

			client := NewRulesClient(server.URL)
			_, _, err := client.ListSchemas(testToken, tt.page, tt.limit)
			assert.NoError(t, err)
		})
	}
}

func TestGetSchema_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/schemas/rule-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": DetectionSchema{
				Type: "detection-schema",
				ID:   "rule-123",
				Attributes: DetectionSchemaAttributes{
					VersionID: "v1",
					Model: map[string]interface{}{
						"correlation": "event_count",
						"filter":      "class_uid:3002",
					},
					View: map[string]interface{}{
						"name":        "Brute Force Attack",
						"description": "Multiple failed logins",
						"severity":    "high",
					},
					Controller: map[string]interface{}{
						"enabled": true,
					},
					CreatedAt: time.Now(),
				},
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	schema, err := client.GetSchema(testToken, "rule-123")

	require.NoError(t, err)
	assert.Equal(t, "rule-123", schema.ID)
	assert.Equal(t, "Brute Force Attack", schema.Attributes.View["name"])
	assert.Equal(t, "high", schema.Attributes.View["severity"])
}

func TestGetSchema_NotFound(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(JSONAPIError{
			Errors: []struct {
				Status string `json:"status"`
				Code   string `json:"code"`
				Title  string `json:"title"`
				Detail string `json:"detail"`
			}{
				{
					Status: "404",
					Code:   "NOT_FOUND",
					Title:  "Schema not found",
					Detail: "Schema with ID 'nonexistent' does not exist",
				},
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	schema, err := client.GetSchema(testToken, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Contains(t, err.Error(), "Schema not found")
}

func TestCreateSchema_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/schemas", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		data := payload["data"].(map[string]interface{})
		assert.Equal(t, "detection-schema", data["type"])

		attrs := data["attributes"].(map[string]interface{})
		model := attrs["model"].(map[string]interface{})
		assert.Equal(t, "value_count", model["correlation"])

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": DetectionSchema{
				Type: "detection-schema",
				ID:   "new-rule-id",
				Attributes: DetectionSchemaAttributes{
					VersionID:  "v1",
					Model:      model,
					View:       attrs["view"].(map[string]interface{}),
					Controller: attrs["controller"].(map[string]interface{}),
					CreatedAt:  time.Now(),
				},
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	model := map[string]interface{}{
		"correlation": "value_count",
		"filter":      "class_uid:4001",
	}
	view := map[string]interface{}{
		"name":     "Port Scan Detection",
		"severity": "medium",
	}
	controller := map[string]interface{}{
		"enabled": true,
	}

	schema, err := client.CreateSchema(testToken, model, view, controller)

	require.NoError(t, err)
	assert.Equal(t, "new-rule-id", schema.ID)
	assert.Equal(t, "value_count", schema.Attributes.Model["correlation"])
}

func TestCreateSchema_ValidationError(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(JSONAPIError{
			Errors: []struct {
				Status string `json:"status"`
				Code   string `json:"code"`
				Title  string `json:"title"`
				Detail string `json:"detail"`
			}{
				{
					Status: "400",
					Code:   "VALIDATION_ERROR",
					Title:  "Invalid schema",
					Detail: "Model correlation type is required",
				},
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	schema, err := client.CreateSchema(testToken, map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{})

	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Contains(t, err.Error(), "Invalid schema")
	assert.Contains(t, err.Error(), "Model correlation type is required")
}

func TestDisableSchema_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/schemas/rule-456/disable", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	err := client.DisableSchema(testToken, "rule-456")

	assert.NoError(t, err)
}

func TestDisableSchema_NotFound(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(JSONAPIError{
			Errors: []struct {
				Status string `json:"status"`
				Code   string `json:"code"`
				Title  string `json:"title"`
				Detail string `json:"detail"`
			}{
				{
					Status: "404",
					Title:  "Not found",
					Detail: "Schema does not exist",
				},
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	err := client.DisableSchema(testToken, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Not found")
}

func TestEnableSchema_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/schemas/rule-789/enable", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	err := client.EnableSchema(testToken, "rule-789")

	assert.NoError(t, err)
}

func TestEnableSchema_AlreadyEnabled(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(JSONAPIError{
			Errors: []struct {
				Status string `json:"status"`
				Code   string `json:"code"`
				Title  string `json:"title"`
				Detail string `json:"detail"`
			}{
				{
					Status: "400",
					Title:  "Already enabled",
					Detail: "Schema is already enabled",
				},
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	err := client.EnableSchema(testToken, "rule-789")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Already enabled")
}

func TestGetVersionHistory_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/schemas/rule-abc/versions", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []DetectionSchema{
				{
					Type: "detection-schema",
					ID:   "rule-abc",
					Attributes: DetectionSchemaAttributes{
						VersionID: "v3",
						Model:     map[string]interface{}{"version": 3},
						CreatedAt: time.Now(),
					},
				},
				{
					Type: "detection-schema",
					ID:   "rule-abc",
					Attributes: DetectionSchemaAttributes{
						VersionID: "v2",
						Model:     map[string]interface{}{"version": 2},
						CreatedAt: time.Now().Add(-24 * time.Hour),
					},
				},
				{
					Type: "detection-schema",
					ID:   "rule-abc",
					Attributes: DetectionSchemaAttributes{
						VersionID: "v1",
						Model:     map[string]interface{}{"version": 1},
						CreatedAt: time.Now().Add(-48 * time.Hour),
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	versions, err := client.GetVersionHistory(testToken, "rule-abc")

	require.NoError(t, err)
	assert.Len(t, versions, 3)
	assert.Equal(t, "v3", versions[0].Attributes.VersionID)
	assert.Equal(t, "v2", versions[1].Attributes.VersionID)
	assert.Equal(t, "v1", versions[2].Attributes.VersionID)
}

func TestGetVersionHistory_NoVersions(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []DetectionSchema{},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)
	versions, err := client.GetVersionHistory(testToken, "rule-new")

	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestRulesClient_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(JSONAPIError{
			Errors: []struct {
				Status string `json:"status"`
				Code   string `json:"code"`
				Title  string `json:"title"`
				Detail string `json:"detail"`
			}{
				{
					Status: "401",
					Code:   "UNAUTHORIZED",
					Title:  "Unauthorized",
					Detail: "Invalid or expired token",
				},
			},
		})
	}))
	defer server.Close()

	client := NewRulesClient(server.URL)

	// Test all methods with bad token
	_, _, err := client.ListSchemas("bad-token", 1, 10)
	assert.Error(t, err)

	_, err = client.GetSchema("bad-token", "rule-1")
	assert.Error(t, err)

	_, err = client.CreateSchema("bad-token", nil, nil, nil)
	assert.Error(t, err)

	err = client.DisableSchema("bad-token", "rule-1")
	assert.Error(t, err)

	err = client.EnableSchema("bad-token", "rule-1")
	assert.Error(t, err)

	_, err = client.GetVersionHistory("bad-token", "rule-1")
	assert.Error(t, err)
}

func TestRulesClient_NetworkError(t *testing.T) {
	client := NewRulesClient("http://invalid-host-does-not-exist.local:99999")

	_, _, err := client.ListSchemas("token", 1, 10)
	assert.Error(t, err)

	_, err = client.GetSchema("token", "rule-1")
	assert.Error(t, err)
}

// Helper function for parsing query params
func mustAtoi(s string) int {
	var result int
	for _, c := range s {
		result = result*10 + int(c-'0')
	}
	return result
}
