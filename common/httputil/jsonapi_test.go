package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONAPIResource(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		id           string
		attributes   interface{}
		status       int
		description  string
	}{
		{
			name:         "Simple user resource",
			resourceType: "user",
			id:           "123",
			attributes: map[string]interface{}{
				"username": "john_doe",
				"email":    "john@example.com",
			},
			status:      http.StatusOK,
			description: "Should write valid JSON:API resource",
		},
		{
			name:         "Resource with nil attributes",
			resourceType: "post",
			id:           "456",
			attributes:   nil,
			status:       http.StatusOK,
			description:  "Should handle nil attributes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSONAPIResource(w, tt.status, tt.resourceType, tt.id, tt.attributes)

			// Check status code
			if w.Code != tt.status {
				t.Errorf("Status code = %d, want %d", w.Code, tt.status)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/vnd.api+json" {
				t.Errorf("Content-Type = %s, want application/vnd.api+json", contentType)
			}

			// Parse response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Verify structure
			data, ok := response["data"].(map[string]interface{})
			if !ok {
				t.Fatal("Response missing 'data' field")
			}

			if data["type"] != tt.resourceType {
				t.Errorf("Resource type = %v, want %s", data["type"], tt.resourceType)
			}

			if data["id"] != tt.id {
				t.Errorf("Resource id = %v, want %s", data["id"], tt.id)
			}
		})
	}
}

func TestWriteJSONAPICollection(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		items        []map[string]interface{}
		pagination   *Pagination
		status       int
		description  string
	}{
		{
			name:         "Collection with pagination",
			resourceType: "user",
			items: []map[string]interface{}{
				{"id": "1", "attributes": map[string]string{"username": "alice"}},
				{"id": "2", "attributes": map[string]string{"username": "bob"}},
			},
			pagination: &Pagination{
				Page:  1,
				Limit: 50,
				Total: 100,
			},
			status:      http.StatusOK,
			description: "Should write collection with pagination meta",
		},
		{
			name:         "Empty collection",
			resourceType: "post",
			items:        []map[string]interface{}{},
			pagination:   nil,
			status:       http.StatusOK,
			description:  "Should handle empty collection",
		},
		{
			name:         "Collection without pagination",
			resourceType: "comment",
			items: []map[string]interface{}{
				{"id": "1", "attributes": map[string]string{"text": "Great!"}},
			},
			pagination:  nil,
			status:      http.StatusOK,
			description: "Should work without pagination",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSONAPICollection(w, tt.status, tt.resourceType, tt.items, tt.pagination)

			// Check status code
			if w.Code != tt.status {
				t.Errorf("Status code = %d, want %d", w.Code, tt.status)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/vnd.api+json" {
				t.Errorf("Content-Type = %s, want application/vnd.api+json", contentType)
			}

			// Parse response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Verify data array
			data, ok := response["data"].([]interface{})
			if !ok {
				t.Fatal("Response missing 'data' array")
			}

			if len(data) != len(tt.items) {
				t.Errorf("Data length = %d, want %d", len(data), len(tt.items))
			}

			// Verify pagination meta if provided
			if tt.pagination != nil {
				meta, ok := response["meta"].(map[string]interface{})
				if !ok {
					t.Fatal("Response missing 'meta' field")
				}

				if int(meta["page"].(float64)) != tt.pagination.Page {
					t.Errorf("Meta page = %v, want %d", meta["page"], tt.pagination.Page)
				}
			}
		})
	}
}

func TestNewJSONAPIError(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		code        string
		title       string
		detail      string
		description string
	}{
		{
			name:        "Validation error",
			status:      http.StatusBadRequest,
			code:        "validation_failed",
			title:       "Validation Failed",
			detail:      "Username is required",
			description: "Should create validation error object",
		},
		{
			name:        "Not found error",
			status:      http.StatusNotFound,
			code:        "not_found",
			title:       "Resource Not Found",
			detail:      "User with ID '123' not found",
			description: "Should create not found error object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewJSONAPIError(tt.status, tt.code, tt.title, tt.detail)

			if err.Status != tt.status {
				t.Errorf("Error status = %d, want %d", err.Status, tt.status)
			}
			if err.Code != tt.code {
				t.Errorf("Error code = %s, want %s", err.Code, tt.code)
			}
			if err.Title != tt.title {
				t.Errorf("Error title = %s, want %s", err.Title, tt.title)
			}
			if err.Detail != tt.detail {
				t.Errorf("Error detail = %s, want %s", err.Detail, tt.detail)
			}
		})
	}
}

func TestWriteJSONAPIValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONAPIValidationError(w, "Username is required")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errors, ok := response["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Fatal("Response missing 'errors' array")
	}

	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "validation_failed" {
		t.Errorf("Error code = %v, want validation_failed", firstError["code"])
	}
}

func TestWriteJSONAPINotFoundError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONAPINotFoundError(w, "user", "123")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusNotFound)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errors, ok := response["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Fatal("Response missing 'errors' array")
	}

	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "not_found" {
		t.Errorf("Error code = %v, want not_found", firstError["code"])
	}

	detail := firstError["detail"].(string)
	if detail != "The requested user with ID '123' was not found" {
		t.Errorf("Error detail = %s, unexpected format", detail)
	}
}

func TestWriteJSONAPIUnauthorizedError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONAPIUnauthorizedError(w, "Invalid token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errors, ok := response["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Fatal("Response missing 'errors' array")
	}

	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "unauthorized" {
		t.Errorf("Error code = %v, want unauthorized", firstError["code"])
	}
}

func TestWriteJSONAPIForbiddenError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONAPIForbiddenError(w, "Admin role required")

	if w.Code != http.StatusForbidden {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusForbidden)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errors, ok := response["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Fatal("Response missing 'errors' array")
	}

	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "forbidden" {
		t.Errorf("Error code = %v, want forbidden", firstError["code"])
	}
}

func TestWriteJSONAPIInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONAPIInternalError(w, "Database connection failed")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	errors, ok := response["errors"].([]interface{})
	if !ok || len(errors) == 0 {
		t.Fatal("Response missing 'errors' array")
	}

	firstError := errors[0].(map[string]interface{})
	if firstError["code"] != "internal_error" {
		t.Errorf("Error code = %v, want internal_error", firstError["code"])
	}
}

func TestBuildJSONAPIResponse(t *testing.T) {
	resource := JSONAPIResource{
		Type: "user",
		ID:   "123",
		Attributes: map[string]string{
			"username": "alice",
		},
	}

	included := []JSONAPIResource{
		{
			Type: "post",
			ID:   "456",
			Attributes: map[string]string{
				"title": "My Post",
			},
		},
	}

	response := BuildJSONAPIResponse(resource, included)

	// Verify data field
	if response["data"] == nil {
		t.Error("Response missing 'data' field")
	}

	// Verify included field
	includedData, ok := response["included"].([]JSONAPIResource)
	if !ok {
		t.Fatal("Response 'included' field has wrong type")
	}

	if len(includedData) != 1 {
		t.Errorf("Included length = %d, want 1", len(includedData))
	}
}

func TestBuildJSONAPIResponse_NoIncluded(t *testing.T) {
	resource := JSONAPIResource{
		Type: "user",
		ID:   "123",
		Attributes: map[string]string{
			"username": "alice",
		},
	}

	response := BuildJSONAPIResponse(resource, nil)

	// Verify data field exists
	if response["data"] == nil {
		t.Error("Response missing 'data' field")
	}

	// Verify included field is not present
	if _, exists := response["included"]; exists {
		t.Error("Response should not have 'included' field when not provided")
	}
}

// Benchmark tests
func BenchmarkWriteJSONAPIResource(b *testing.B) {
	attributes := map[string]interface{}{
		"username": "john_doe",
		"email":    "john@example.com",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		WriteJSONAPIResource(w, http.StatusOK, "user", "123", attributes)
	}
}

func BenchmarkWriteJSONAPICollection(b *testing.B) {
	items := []map[string]interface{}{
		{"id": "1", "attributes": map[string]string{"username": "alice"}},
		{"id": "2", "attributes": map[string]string{"username": "bob"}},
	}
	pagination := &Pagination{Page: 1, Limit: 50, Total: 100}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		WriteJSONAPICollection(w, http.StatusOK, "user", items, pagination)
	}
}
