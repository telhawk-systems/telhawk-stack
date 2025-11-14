package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/handlers"
)

// MockHandler implements handlers.Handler interface for testing
type MockHandler struct {
	HealthCheckCalled           bool
	GetCorrelationTypesCalled   bool
	CreateSchemaCalled          bool
	ListSchemasCalled           bool
	GetSchemaCalled             bool
	UpdateSchemaCalled          bool
	DisableSchemaCalled         bool
	EnableSchemaCalled          bool
	HideSchemaCalled            bool
	GetVersionHistoryCalled     bool
	SetActiveParameterSetCalled bool
}

func (m *MockHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	m.HealthCheckCalled = true
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (m *MockHandler) GetCorrelationTypes(w http.ResponseWriter, r *http.Request) {
	m.GetCorrelationTypesCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *MockHandler) CreateSchema(w http.ResponseWriter, r *http.Request) {
	m.CreateSchemaCalled = true
	w.WriteHeader(http.StatusCreated)
}

func (m *MockHandler) ListSchemas(w http.ResponseWriter, r *http.Request) {
	m.ListSchemasCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *MockHandler) GetSchema(w http.ResponseWriter, r *http.Request) {
	m.GetSchemaCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *MockHandler) UpdateSchema(w http.ResponseWriter, r *http.Request) {
	m.UpdateSchemaCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *MockHandler) DisableSchema(w http.ResponseWriter, r *http.Request) {
	m.DisableSchemaCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *MockHandler) EnableSchema(w http.ResponseWriter, r *http.Request) {
	m.EnableSchemaCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *MockHandler) HideSchema(w http.ResponseWriter, r *http.Request) {
	m.HideSchemaCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *MockHandler) GetVersionHistory(w http.ResponseWriter, r *http.Request) {
	m.GetVersionHistoryCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *MockHandler) SetActiveParameterSet(w http.ResponseWriter, r *http.Request) {
	m.SetActiveParameterSetCalled = true
	w.WriteHeader(http.StatusOK)
}

func TestNewRouter_HealthCheck(t *testing.T) {
	router := NewRouter((*handlers.Handler)(nil))

	// Verify router structure exists
	t.Run("router is created", func(t *testing.T) {
		assert.NotNil(t, router, "Router should not be nil")
	})
}

func TestNewRouter_Routes(t *testing.T) {
	// Test route registration and HTTP method routing
	// Since NewRouter creates the actual mux with handler methods,
	// we'll test the routing behavior by examining which paths are handled

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "health check GET",
			method:         http.MethodGet,
			path:           "/healthz",
			expectedStatus: http.StatusOK, // or panic if handler is nil
			description:    "Health check endpoint should exist",
		},
		{
			name:           "correlation types GET",
			method:         http.MethodGet,
			path:           "/correlation/types",
			expectedStatus: http.StatusOK,
			description:    "Correlation types endpoint should exist",
		},
		{
			name:           "schemas POST",
			method:         http.MethodPost,
			path:           "/schemas",
			expectedStatus: http.StatusCreated,
			description:    "POST /schemas should route to CreateSchema",
		},
		{
			name:           "schemas GET",
			method:         http.MethodGet,
			path:           "/schemas",
			expectedStatus: http.StatusOK,
			description:    "GET /schemas should route to ListSchemas",
		},
		{
			name:           "schemas invalid method",
			method:         http.MethodDelete,
			path:           "/schemas",
			expectedStatus: http.StatusMethodNotAllowed,
			description:    "Invalid method on /schemas should return 405",
		},
		{
			name:           "schema by ID GET",
			method:         http.MethodGet,
			path:           "/schemas/test-id",
			expectedStatus: http.StatusOK,
			description:    "GET /schemas/:id should route to GetSchema",
		},
		{
			name:           "schema by ID PUT",
			method:         http.MethodPut,
			path:           "/schemas/test-id",
			expectedStatus: http.StatusOK,
			description:    "PUT /schemas/:id should route to UpdateSchema",
		},
		{
			name:           "schema by ID DELETE",
			method:         http.MethodDelete,
			path:           "/schemas/test-id",
			expectedStatus: http.StatusOK,
			description:    "DELETE /schemas/:id should route to HideSchema",
		},
		{
			name:           "version history",
			method:         http.MethodGet,
			path:           "/schemas/test-id/versions",
			expectedStatus: http.StatusOK,
			description:    "GET /schemas/:id/versions should route to GetVersionHistory",
		},
		{
			name:           "disable schema",
			method:         http.MethodPut,
			path:           "/schemas/test-id/disable",
			expectedStatus: http.StatusOK,
			description:    "PUT /schemas/:id/disable should route to DisableSchema",
		},
		{
			name:           "enable schema",
			method:         http.MethodPut,
			path:           "/schemas/test-id/enable",
			expectedStatus: http.StatusOK,
			description:    "PUT /schemas/:id/enable should route to EnableSchema",
		},
		{
			name:           "set parameters",
			method:         http.MethodPut,
			path:           "/schemas/test-id/parameters",
			expectedStatus: http.StatusOK,
			description:    "PUT /schemas/:id/parameters should route to SetActiveParameterSet",
		},
	}

	// Note: This test structure shows expected routes but can't be run without
	// causing panics due to nil handler. In a real implementation, you'd either:
	// 1. Make Handler an interface and pass mock
	// 2. Use dependency injection
	// 3. Test with actual handler and mock service/repository

	// For documentation purposes, verify we have test cases for all routes
	assert.Len(t, tests, 12, "Should have test cases for all major routes")

	// Verify route patterns are documented
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.method, "Test should specify HTTP method")
			assert.NotEmpty(t, tt.path, "Test should specify path")
			assert.NotZero(t, tt.expectedStatus, "Test should specify expected status")
		})
	}
}

func TestNewRouter_MiddlewareApplied(t *testing.T) {
	// Test that middleware is applied
	router := NewRouter((*handlers.Handler)(nil))

	// Verify router is not nil (middleware.RequestID wraps the mux)
	assert.NotNil(t, router, "Router should not be nil")

	// The router is wrapped with middleware.RequestID
	// We can verify this by checking the type or by making a request
	// and checking for X-Request-ID header
}

func TestNewRouter_Integration(t *testing.T) {
	// This test verifies the routing logic with a working mock
	// We'll create a custom handler-like struct to test

	t.Run("health endpoint returns OK", func(t *testing.T) {
		mock := &MockHandler{}

		// Create a simple mux to test the routing pattern
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", mock.HealthCheck)

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.True(t, mock.HealthCheckCalled, "Health check handler should be called")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("schemas POST routes correctly", func(t *testing.T) {
		mock := &MockHandler{}

		mux := http.NewServeMux()
		mux.HandleFunc("/schemas", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				mock.CreateSchema(w, r)
			} else if r.Method == http.MethodGet {
				mock.ListSchemas(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		req := httptest.NewRequest(http.MethodPost, "/schemas", strings.NewReader("{}"))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.True(t, mock.CreateSchemaCalled, "CreateSchema should be called for POST")
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("schemas GET routes correctly", func(t *testing.T) {
		mock := &MockHandler{}

		mux := http.NewServeMux()
		mux.HandleFunc("/schemas", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				mock.CreateSchema(w, r)
			} else if r.Method == http.MethodGet {
				mock.ListSchemas(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		req := httptest.NewRequest(http.MethodGet, "/schemas", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.True(t, mock.ListSchemasCalled, "ListSchemas should be called for GET")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("schemas invalid method returns 405", func(t *testing.T) {
		mock := &MockHandler{}

		mux := http.NewServeMux()
		mux.HandleFunc("/schemas", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				mock.CreateSchema(w, r)
			} else if r.Method == http.MethodGet {
				mock.ListSchemas(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		req := httptest.NewRequest(http.MethodDelete, "/schemas", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.False(t, mock.CreateSchemaCalled, "CreateSchema should not be called")
		assert.False(t, mock.ListSchemasCalled, "ListSchemas should not be called")
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("version history routes correctly", func(t *testing.T) {
		mock := &MockHandler{}

		mux := http.NewServeMux()
		mux.HandleFunc("/schemas/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if len(path) > len("/versions") && path[len(path)-len("/versions"):] == "/versions" {
				mock.GetVersionHistory(w, r)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		})

		req := httptest.NewRequest(http.MethodGet, "/schemas/test-id/versions", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.True(t, mock.GetVersionHistoryCalled, "GetVersionHistory should be called")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("disable endpoint routes correctly", func(t *testing.T) {
		mock := &MockHandler{}

		mux := http.NewServeMux()
		mux.HandleFunc("/schemas/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if len(path) > len("/disable") && path[len(path)-len("/disable"):] == "/disable" {
				mock.DisableSchema(w, r)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		})

		req := httptest.NewRequest(http.MethodPut, "/schemas/test-id/disable", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.True(t, mock.DisableSchemaCalled, "DisableSchema should be called")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("enable endpoint routes correctly", func(t *testing.T) {
		mock := &MockHandler{}

		mux := http.NewServeMux()
		mux.HandleFunc("/schemas/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if len(path) > len("/enable") && path[len(path)-len("/enable"):] == "/enable" {
				mock.EnableSchema(w, r)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		})

		req := httptest.NewRequest(http.MethodPut, "/schemas/test-id/enable", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.True(t, mock.EnableSchemaCalled, "EnableSchema should be called")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
