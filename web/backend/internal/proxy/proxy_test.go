package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/auth"
)

func TestProxy_Handler_BasicProxying(t *testing.T) {
	// Create mock backend service
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	expectedBody := `{"message": "success"}`
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestProxy_Handler_PreservesPath(t *testing.T) {
	var receivedPath string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	expectedPath := "/api/v1/users/123"
	if receivedPath != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, receivedPath)
	}
}

func TestProxy_Handler_PreservesQueryString(t *testing.T) {
	var receivedQuery string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/search?q=test&limit=10", nil)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	expectedQuery := "q=test&limit=10"
	if receivedQuery != expectedQuery {
		t.Errorf("Expected query '%s', got '%s'", expectedQuery, receivedQuery)
	}
}

func TestProxy_Handler_PreservesMethod(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var receivedMethod string

			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			mockAuthClient := &auth.Client{}
			proxy := NewProxy(backend.URL, mockAuthClient)

			var body io.Reader
			if method != "GET" {
				body = strings.NewReader(`{"test": "data"}`)
			}

			req := httptest.NewRequest(method, "/api/v1/test", body)
			rr := httptest.NewRecorder()

			proxy.Handler().ServeHTTP(rr, req)

			if receivedMethod != method {
				t.Errorf("Expected method '%s', got '%s'", method, receivedMethod)
			}
		})
	}
}

func TestProxy_Handler_PreservesBody(t *testing.T) {
	var receivedBody string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBody = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	expectedBody := `{"username": "admin", "password": "secret"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(expectedBody))
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	if receivedBody != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, receivedBody)
	}
}

func TestProxy_Handler_PreservesHeaders(t *testing.T) {
	var receivedHeaders http.Header

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Custom-Header", "custom-value")
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header to be preserved")
	}
	if receivedHeaders.Get("Accept") != "application/json" {
		t.Errorf("Expected Accept header to be preserved")
	}
	if receivedHeaders.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("Expected X-Custom-Header to be preserved")
	}
}

func TestProxy_Handler_InjectsAuthorizationFromCookie(t *testing.T) {
	var receivedAuthHeader string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "test-token-123"})
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	expectedAuth := "Bearer test-token-123"
	if receivedAuthHeader != expectedAuth {
		t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, receivedAuthHeader)
	}
}

func TestProxy_Handler_DoesNotOverrideExistingAuthorization(t *testing.T) {
	var receivedAuthHeader string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer existing-token")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-token"})
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	expectedAuth := "Bearer existing-token"
	if receivedAuthHeader != expectedAuth {
		t.Errorf("Expected existing Authorization header to be preserved, got '%s'", receivedAuthHeader)
	}
}

func TestProxy_Handler_InjectsUserIDHeader(t *testing.T) {
	var receivedUserID string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserID = r.Header.Get("X-User-ID")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	ctx := context.WithValue(req.Context(), auth.UserIDKey, "user-123")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	if receivedUserID != "user-123" {
		t.Errorf("Expected X-User-ID 'user-123', got '%s'", receivedUserID)
	}
}

func TestProxy_Handler_InjectsRolesHeader(t *testing.T) {
	var receivedRoles string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRoles = r.Header.Get("X-User-Roles")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	ctx := context.WithValue(req.Context(), auth.RolesKey, []string{"admin", "user", "moderator"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	expectedRoles := "admin,user,moderator"
	if receivedRoles != expectedRoles {
		t.Errorf("Expected X-User-Roles '%s', got '%s'", expectedRoles, receivedRoles)
	}
}

func TestProxy_Handler_NoRolesHeaderWhenEmpty(t *testing.T) {
	var receivedRoles string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRoles = r.Header.Get("X-User-Roles")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	if receivedRoles != "" {
		t.Errorf("Expected no X-User-Roles header, got '%s'", receivedRoles)
	}
}

func TestProxy_Handler_ForwardsResponseHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Response", "custom-value")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type header to be forwarded")
	}
	if customHeader := rr.Header().Get("X-Custom-Response"); customHeader != "custom-value" {
		t.Errorf("Expected X-Custom-Response header to be forwarded")
	}
	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Errorf("Expected Cache-Control header to be forwarded")
	}
}

func TestProxy_Handler_ForwardsResponseStatus(t *testing.T) {
	tests := []struct {
		name           string
		backendStatus  int
		expectedStatus int
	}{
		{"Success", http.StatusOK, http.StatusOK},
		{"Created", http.StatusCreated, http.StatusCreated},
		{"No Content", http.StatusNoContent, http.StatusNoContent},
		{"Bad Request", http.StatusBadRequest, http.StatusBadRequest},
		{"Unauthorized", http.StatusUnauthorized, http.StatusUnauthorized},
		{"Forbidden", http.StatusForbidden, http.StatusForbidden},
		{"Not Found", http.StatusNotFound, http.StatusNotFound},
		{"Internal Server Error", http.StatusInternalServerError, http.StatusInternalServerError},
		{"Service Unavailable", http.StatusServiceUnavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.backendStatus)
			}))
			defer backend.Close()

			mockAuthClient := &auth.Client{}
			proxy := NewProxy(backend.URL, mockAuthClient)

			req := httptest.NewRequest("GET", "/api/v1/test", nil)
			rr := httptest.NewRecorder()

			proxy.Handler().ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestProxy_Handler_ForwardsResponseBody(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "test response", "count": 42}`))
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	expectedBody := `{"data": "test response", "count": 42}`
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestProxy_Handler_BackendUnavailable(t *testing.T) {
	// Use an invalid URL to simulate backend unavailability
	mockAuthClient := &auth.Client{}
	proxy := NewProxy("http://localhost:99999", mockAuthClient)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rr.Code)
	}

	expectedBody := "Service unavailable\n"
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, body)
	}
}

func TestProxy_Handler_CompleteFlow(t *testing.T) {
	var (
		receivedMethod  string
		receivedPath    string
		receivedQuery   string
		receivedBody    string
		receivedAuth    string
		receivedUserID  string
		receivedRoles   string
		receivedHeaders http.Header
	)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBody = string(bodyBytes)
		receivedAuth = r.Header.Get("Authorization")
		receivedUserID = r.Header.Get("X-User-ID")
		receivedRoles = r.Header.Get("X-User-Roles")
		receivedHeaders = r.Header

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Response-ID", "12345")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": "12345", "status": "created"}`))
	}))
	defer backend.Close()

	mockAuthClient := &auth.Client{}
	proxy := NewProxy(backend.URL, mockAuthClient)

	requestBody := `{"name": "Test Item"}`
	req := httptest.NewRequest("POST", "/api/v1/items?category=test", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "my-token"})

	ctx := context.WithValue(req.Context(), auth.UserIDKey, "user-456")
	ctx = context.WithValue(ctx, auth.RolesKey, []string{"admin", "editor"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	proxy.Handler().ServeHTTP(rr, req)

	// Verify request forwarding
	if receivedMethod != "POST" {
		t.Errorf("Expected method POST, got %s", receivedMethod)
	}
	if receivedPath != "/api/v1/items" {
		t.Errorf("Expected path /api/v1/items, got %s", receivedPath)
	}
	if receivedQuery != "category=test" {
		t.Errorf("Expected query category=test, got %s", receivedQuery)
	}
	if receivedBody != requestBody {
		t.Errorf("Expected body %s, got %s", requestBody, receivedBody)
	}
	if receivedAuth != "Bearer my-token" {
		t.Errorf("Expected Authorization 'Bearer my-token', got %s", receivedAuth)
	}
	if receivedUserID != "user-456" {
		t.Errorf("Expected X-User-ID 'user-456', got %s", receivedUserID)
	}
	if receivedRoles != "admin,editor" {
		t.Errorf("Expected X-User-Roles 'admin,editor', got %s", receivedRoles)
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type header to be preserved")
	}

	// Verify response forwarding
	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rr.Code)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
	if responseID := rr.Header().Get("X-Response-ID"); responseID != "12345" {
		t.Errorf("Expected X-Response-ID 12345, got %s", responseID)
	}

	expectedRespBody := `{"id": "12345", "status": "created"}`
	if body := rr.Body.String(); body != expectedRespBody {
		t.Errorf("Expected body %s, got %s", expectedRespBody, body)
	}
}
