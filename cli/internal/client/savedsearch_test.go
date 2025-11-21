package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavedSearchCreate_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/saved-searches", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.api+json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		data := payload["data"].(map[string]interface{})
		attrs := data["attributes"].(map[string]interface{})
		assert.Equal(t, "My Search", attrs["name"])
		assert.Equal(t, true, attrs["is_global"])

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"type": "saved-search",
				"id":   "search-123",
				"attributes": map[string]interface{}{
					"version_id":  "v1",
					"name":        "My Search",
					"query":       attrs["query"],
					"is_global":   true,
					"created_at":  "2024-11-20T10:00:00Z",
					"disabled_at": nil,
					"hidden_at":   nil,
				},
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	query := map[string]interface{}{"q": "severity:high"}
	filters := map[string]interface{}{"time_range": "1h"}

	search, err := client.SavedSearchCreate(testToken, server.URL, "My Search", query, filters, true)

	require.NoError(t, err)
	assert.Equal(t, "search-123", search.ID)
	assert.Equal(t, "v1", search.VersionID)
	assert.Equal(t, "My Search", search.Name)
	assert.True(t, search.IsGlobal)
	assert.NotNil(t, search.Query)
}

func TestSavedSearchCreate_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	query := map[string]interface{}{"q": "test"}

	search, err := client.SavedSearchCreate("bad-token", server.URL, "Test", query, nil, false)

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "create failed")
}

func TestSavedSearchList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/saved-searches", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/vnd.api+json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"type": "saved-search",
					"id":   "search-1",
					"attributes": map[string]interface{}{
						"name":       "Search 1",
						"query":      map[string]interface{}{"q": "test"},
						"is_global":  true,
						"created_at": "2024-11-20T10:00:00Z",
					},
				},
				{
					"type": "saved-search",
					"id":   "search-2",
					"attributes": map[string]interface{}{
						"name":       "Search 2",
						"query":      map[string]interface{}{"q": "error"},
						"is_global":  false,
						"created_at": "2024-11-20T11:00:00Z",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	searches, err := client.SavedSearchList(server.URL, false)

	require.NoError(t, err)
	assert.Len(t, searches, 2)
	assert.Equal(t, "search-1", searches[0].ID)
	assert.Equal(t, "Search 1", searches[0].Name)
	assert.True(t, searches[0].IsGlobal)
	assert.Equal(t, "search-2", searches[1].ID)
	assert.Equal(t, "Search 2", searches[1].Name)
	assert.False(t, searches[1].IsGlobal)
}

func TestSavedSearchList_WithShowAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.URL.Query().Get("filter[show_all]"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	_, err := client.SavedSearchList(server.URL, true)

	assert.NoError(t, err)
}

func TestSavedSearchList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	searches, err := client.SavedSearchList(server.URL, false)

	require.NoError(t, err)
	assert.Empty(t, searches)
}

func TestSavedSearchGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/saved-searches/search-456", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"type": "saved-search",
				"id":   "search-456",
				"attributes": map[string]interface{}{
					"version_id": "v2",
					"name":       "Specific Search",
					"query":      map[string]interface{}{"q": "severity:critical"},
					"is_global":  false,
					"created_at": "2024-11-20T12:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	search, err := client.SavedSearchGet(server.URL, "search-456")

	require.NoError(t, err)
	assert.Equal(t, "search-456", search.ID)
	assert.Equal(t, "v2", search.VersionID)
	assert.Equal(t, "Specific Search", search.Name)
	assert.False(t, search.IsGlobal)
}

func TestSavedSearchGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	search, err := client.SavedSearchGet(server.URL, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "get failed")
}

func TestSavedSearchUpdate_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/saved-searches/search-789", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))

		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		data := payload["data"].(map[string]interface{})
		attrs := data["attributes"].(map[string]interface{})
		assert.Equal(t, "Updated Name", attrs["name"])

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"type": "saved-search",
				"id":   "search-789",
				"attributes": map[string]interface{}{
					"name":       "Updated Name",
					"query":      map[string]interface{}{"q": "test"},
					"is_global":  false,
					"created_at": "2024-11-20T10:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	attrs := map[string]interface{}{"name": "Updated Name"}

	search, err := client.SavedSearchUpdate(testToken, server.URL, "search-789", attrs)

	require.NoError(t, err)
	assert.Equal(t, "search-789", search.ID)
	assert.Equal(t, "Updated Name", search.Name)
}

func TestSavedSearchUpdate_ValidationError(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	attrs := map[string]interface{}{"name": ""}

	search, err := client.SavedSearchUpdate(testToken, server.URL, "search-1", attrs)

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "update failed")
}

func TestSavedSearchAction_Disable(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/saved-searches/search-abc/disable", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"type": "saved-search",
				"id":   "search-abc",
				"attributes": map[string]interface{}{
					"name":        "Test Search",
					"query":       map[string]interface{}{},
					"is_global":   false,
					"created_at":  "2024-11-20T10:00:00Z",
					"disabled_at": "2024-11-20T15:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	search, err := client.SavedSearchAction(testToken, server.URL, "search-abc", "disable")

	require.NoError(t, err)
	assert.Equal(t, "search-abc", search.ID)
	assert.NotNil(t, search.DisabledAt)
}

func TestSavedSearchAction_Enable(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/saved-searches/search-def/enable", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"type": "saved-search",
				"id":   "search-def",
				"attributes": map[string]interface{}{
					"name":        "Test",
					"query":       map[string]interface{}{},
					"created_at":  "2024-11-20T10:00:00Z",
					"disabled_at": nil,
				},
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	search, err := client.SavedSearchAction(testToken, server.URL, "search-def", "enable")

	require.NoError(t, err)
	assert.Equal(t, "search-def", search.ID)
	assert.Nil(t, search.DisabledAt)
}

func TestSavedSearchAction_Hide(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/saved-searches/search-xyz/hide", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"type": "saved-search",
				"id":   "search-xyz",
				"attributes": map[string]interface{}{
					"name":       "Hidden Search",
					"query":      map[string]interface{}{},
					"created_at": "2024-11-20T10:00:00Z",
					"hidden_at":  "2024-11-20T16:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	search, err := client.SavedSearchAction(testToken, server.URL, "search-xyz", "hide")

	require.NoError(t, err)
	assert.NotNil(t, search.HiddenAt)
}

func TestSavedSearchAction_Error(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewQueryClient(server.URL)
	search, err := client.SavedSearchAction(testToken, server.URL, "search-1", "disable")

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "disable failed")
}

func TestSavedFromResource_AllFields(t *testing.T) {
	disabledAt := "2024-11-20T14:00:00Z"
	hiddenAt := "2024-11-20T15:00:00Z"

	resource := &jsonAPIResource{
		Type: "saved-search",
		ID:   "test-123",
		Attributes: map[string]interface{}{
			"version_id":  "v3",
			"name":        "Complete Search",
			"query":       map[string]interface{}{"q": "test"},
			"is_global":   true,
			"created_at":  "2024-11-20T10:00:00Z",
			"disabled_at": disabledAt,
			"hidden_at":   hiddenAt,
		},
	}

	search := savedFromResource(resource)

	assert.Equal(t, "test-123", search.ID)
	assert.Equal(t, "v3", search.VersionID)
	assert.Equal(t, "Complete Search", search.Name)
	assert.True(t, search.IsGlobal)
	assert.Equal(t, "2024-11-20T10:00:00Z", search.CreatedAt)
	assert.NotNil(t, search.DisabledAt)
	assert.Equal(t, disabledAt, *search.DisabledAt)
	assert.NotNil(t, search.HiddenAt)
	assert.Equal(t, hiddenAt, *search.HiddenAt)
}

func TestSavedFromResource_MinimalFields(t *testing.T) {
	resource := &jsonAPIResource{
		Type:       "saved-search",
		ID:         "minimal-456",
		Attributes: map[string]interface{}{},
	}

	search := savedFromResource(resource)

	assert.Equal(t, "minimal-456", search.ID)
	assert.Empty(t, search.VersionID)
	assert.Empty(t, search.Name)
	assert.False(t, search.IsGlobal)
	assert.Nil(t, search.DisabledAt)
	assert.Nil(t, search.HiddenAt)
}

func TestSavedSearchList_NetworkError(t *testing.T) {
	client := NewQueryClient("http://invalid-host-does-not-exist.local:99999")
	searches, err := client.SavedSearchList("http://invalid-host-does-not-exist.local:99999", false)

	assert.Error(t, err)
	assert.Nil(t, searches)
}
