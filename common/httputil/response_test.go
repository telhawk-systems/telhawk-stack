package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           interface{}
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "successful response with map",
			status:         http.StatusOK,
			data:           map[string]string{"message": "success"},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "error response",
			status:         http.StatusBadRequest,
			data:           map[string]string{"error": "bad request"},
			expectedStatus: http.StatusBadRequest,
			expectedType:   "application/json",
		},
		{
			name:           "response with struct",
			status:         http.StatusCreated,
			data:           struct{ ID string }{"123"},
			expectedStatus: http.StatusCreated,
			expectedType:   "application/json",
		},
		{
			name:           "response with slice",
			status:         http.StatusOK,
			data:           []string{"one", "two", "three"},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSON(w, tt.status, tt.data)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("expected content type %q, got %q", tt.expectedType, contentType)
			}

			// Verify JSON is valid
			var result interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Errorf("response is not valid JSON: %v", err)
			}
		})
	}
}

func TestWriteJSON_InvalidData(t *testing.T) {
	// Test with data that cannot be marshaled (e.g., channel)
	w := httptest.NewRecorder()
	invalidData := make(chan int)

	// This should not panic, but will log an error
	WriteJSON(w, http.StatusOK, invalidData)

	// Status should still be set
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Content-Type should still be set
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type to be set")
	}
}

func TestWriteJSONAPI(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		data           interface{}
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "successful JSON:API response",
			status:         http.StatusOK,
			data:           map[string]interface{}{"data": map[string]string{"type": "users", "id": "1"}},
			expectedStatus: http.StatusOK,
			expectedType:   "application/vnd.api+json",
		},
		{
			name:   "JSON:API error response",
			status: http.StatusBadRequest,
			data: map[string]interface{}{
				"errors": []map[string]interface{}{
					{"status": 400, "title": "Bad Request"},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedType:   "application/vnd.api+json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSONAPI(w, tt.status, tt.data)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("expected content type %q, got %q", tt.expectedType, contentType)
			}

			// Verify JSON is valid
			var result interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Errorf("response is not valid JSON: %v", err)
			}
		})
	}
}

func TestWriteJSONAPI_InvalidData(t *testing.T) {
	w := httptest.NewRecorder()
	invalidData := make(chan int)

	WriteJSONAPI(w, http.StatusOK, invalidData)

	// Status should still be set
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Content-Type should be JSON:API
	if w.Header().Get("Content-Type") != "application/vnd.api+json" {
		t.Errorf("expected Content-Type to be application/vnd.api+json")
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		message        string
		expectedStatus int
	}{
		{
			name:           "not found error",
			status:         http.StatusNotFound,
			message:        "resource not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "internal server error",
			status:         http.StatusInternalServerError,
			message:        "internal error",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "validation error",
			status:         http.StatusUnprocessableEntity,
			message:        "validation failed",
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tt.status, tt.message)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json")
			}

			// Parse response
			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			// Verify error message
			if response["error"] != tt.message {
				t.Errorf("expected error message %q, got %q", tt.message, response["error"])
			}
		})
	}
}

func TestWriteJSONAPIError(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		code           string
		title          string
		detail         string
		expectedStatus int
	}{
		{
			name:           "not found error",
			status:         http.StatusNotFound,
			code:           "not_found",
			title:          "Resource Not Found",
			detail:         "The requested resource does not exist",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "validation error",
			status:         http.StatusUnprocessableEntity,
			code:           "validation_error",
			title:          "Validation Failed",
			detail:         "The username field is required",
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSONAPIError(w, tt.status, tt.code, tt.title, tt.detail)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			if w.Header().Get("Content-Type") != "application/vnd.api+json" {
				t.Errorf("expected Content-Type application/vnd.api+json")
			}

			// Parse response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			// Verify errors array exists
			errors, ok := response["errors"].([]interface{})
			if !ok {
				t.Fatal("expected 'errors' array in response")
			}

			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d", len(errors))
			}

			// Verify error structure
			errorObj, ok := errors[0].(map[string]interface{})
			if !ok {
				t.Fatal("error is not an object")
			}

			// Check status (as float64 due to JSON unmarshal)
			if int(errorObj["status"].(float64)) != tt.status {
				t.Errorf("expected status %d, got %v", tt.status, errorObj["status"])
			}

			// Check code
			if errorObj["code"] != tt.code {
				t.Errorf("expected code %q, got %v", tt.code, errorObj["code"])
			}

			// Check title
			if errorObj["title"] != tt.title {
				t.Errorf("expected title %q, got %v", tt.title, errorObj["title"])
			}

			// Check detail
			if errorObj["detail"] != tt.detail {
				t.Errorf("expected detail %q, got %v", tt.detail, errorObj["detail"])
			}
		})
	}
}
