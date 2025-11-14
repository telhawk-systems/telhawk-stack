package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/models"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/service"
)

// MockRepository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateSchema(ctx context.Context, schema *models.DetectionSchema) error {
	args := m.Called(ctx, schema)
	return args.Error(0)
}

func (m *MockRepository) GetSchemaByVersionID(ctx context.Context, versionID string) (*models.DetectionSchema, error) {
	args := m.Called(ctx, versionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DetectionSchema), args.Error(1)
}

func (m *MockRepository) GetLatestSchemaByID(ctx context.Context, id string) (*models.DetectionSchema, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DetectionSchema), args.Error(1)
}

func (m *MockRepository) GetSchemaVersionHistory(ctx context.Context, id string) ([]*models.DetectionSchemaVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DetectionSchemaVersion), args.Error(1)
}

func (m *MockRepository) ListSchemas(ctx context.Context, req *models.ListSchemasRequest) ([]*models.DetectionSchema, int, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.DetectionSchema), args.Int(1), args.Error(2)
}

func (m *MockRepository) DisableSchema(ctx context.Context, versionID, userID string) error {
	args := m.Called(ctx, versionID, userID)
	return args.Error(0)
}

func (m *MockRepository) EnableSchema(ctx context.Context, versionID string) error {
	args := m.Called(ctx, versionID)
	return args.Error(0)
}

func (m *MockRepository) HideSchema(ctx context.Context, versionID, userID string) error {
	args := m.Called(ctx, versionID, userID)
	return args.Error(0)
}

func (m *MockRepository) SetActiveParameterSet(ctx context.Context, versionID, parameterSet string) error {
	args := m.Called(ctx, versionID, parameterSet)
	return args.Error(0)
}

func (m *MockRepository) Close() {}

// Helper to create a test schema
func createTestSchema() *models.DetectionSchema {
	return &models.DetectionSchema{
		ID:        "test-id",
		VersionID: "test-version-id",
		Model: map[string]interface{}{
			"query": "SELECT * FROM events",
		},
		View: map[string]interface{}{
			"title":    "Test Rule",
			"severity": "high",
		},
		Controller: map[string]interface{}{
			"enabled": true,
		},
		CreatedBy: "test-user",
		CreatedAt: time.Now(),
		Version:   1,
	}
}

func TestNewRouter_HealthCheckRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestNewRouter_CorrelationTypesRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/correlation/types", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestNewRouter_SchemasPostRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	schema := createTestSchema()
	mockRepo.On("CreateSchema", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("GetSchemaByVersionID", mock.Anything, mock.Anything).Return(schema, nil)

	reqBody := map[string]interface{}{
		"model":      schema.Model,
		"view":       schema.View,
		"controller": schema.Controller,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/schemas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "test-user")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_SchemasGetRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	schemas := []*models.DetectionSchema{createTestSchema()}
	mockRepo.On("ListSchemas", mock.Anything, mock.Anything).Return(schemas, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/schemas", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_SchemasInvalidMethodRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/schemas", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestNewRouter_GetSchemaByIDRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	schema := createTestSchema()
	mockRepo.On("GetSchemaByVersionID", mock.Anything, "test-version-id").Return(schema, nil)

	req := httptest.NewRequest(http.MethodGet, "/schemas/test-version-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_UpdateSchemaRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	schema := createTestSchema()
	mockRepo.On("GetLatestSchemaByID", mock.Anything, "test-id").Return(schema, nil)
	mockRepo.On("CreateSchema", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("GetSchemaByVersionID", mock.Anything, mock.Anything).Return(schema, nil)

	reqBody := map[string]interface{}{
		"model":      schema.Model,
		"view":       schema.View,
		"controller": schema.Controller,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/schemas/test-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "test-user")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_DeleteSchemaRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	schema := createTestSchema()
	mockRepo.On("GetSchemaByVersionID", mock.Anything, "test-version-id").Return(schema, nil)
	// Note: HideSchema uses hardcoded user ID in handler (TODO: extract from JWT)
	mockRepo.On("HideSchema", mock.Anything, "test-version-id", "00000000-0000-0000-0000-000000000001").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/schemas/test-version-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_VersionHistoryRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	versions := []*models.DetectionSchemaVersion{
		{
			VersionID: "v1",
			Version:   1,
			Title:     "Test Rule",
			CreatedBy: "test-user",
			CreatedAt: time.Now(),
		},
	}
	mockRepo.On("GetSchemaVersionHistory", mock.Anything, "test-id").Return(versions, nil)

	req := httptest.NewRequest(http.MethodGet, "/schemas/test-id/versions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_DisableSchemaRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	schema := createTestSchema()
	mockRepo.On("GetSchemaByVersionID", mock.Anything, "test-version-id").Return(schema, nil)
	mockRepo.On("DisableSchema", mock.Anything, "test-version-id", "00000000-0000-0000-0000-000000000001").Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/schemas/test-version-id/disable", nil)
	req.Header.Set("X-User-ID", "test-user")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_EnableSchemaRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	mockRepo.On("EnableSchema", mock.Anything, "test-version-id").Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/schemas/test-version-id/enable", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_SetActiveParameterSetRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	mockRepo.On("SetActiveParameterSet", mock.Anything, "test-version-id", "default").Return(nil)

	reqBody := map[string]interface{}{
		"active_parameter_set": "default",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/schemas/test-version-id/parameters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestNewRouter_UnknownSchemaRoute(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	// Test an unknown route that doesn't match any patterns
	req := httptest.NewRequest(http.MethodPatch, "/schemas/test-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNewRouter_MiddlewareRequestID(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewService(mockRepo)
	handler := handlers.NewHandler(svc)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// RequestID middleware should add X-Request-ID header
	requestID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID, "RequestID middleware should set X-Request-ID header")
}

func TestNewRouter_AllRoutesWithMiddleware(t *testing.T) {
	// Verify all routes work through the middleware chain
	tests := []struct {
		name           string
		method         string
		path           string
		body           map[string]interface{}
		setupMock      func(*MockRepository)
		expectedStatus int
	}{
		{
			name:   "health check",
			method: http.MethodGet,
			path:   "/healthz",
			setupMock: func(m *MockRepository) {
				// No setup needed
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "correlation types",
			method: http.MethodGet,
			path:   "/correlation/types",
			setupMock: func(m *MockRepository) {
				// No setup needed
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "list schemas",
			method: http.MethodGet,
			path:   "/schemas",
			setupMock: func(m *MockRepository) {
				m.On("ListSchemas", mock.Anything, mock.Anything).Return([]*models.DetectionSchema{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := service.NewService(mockRepo)
			handler := handlers.NewHandler(svc)
			router := NewRouter(handler)

			tt.setupMock(mockRepo)

			var bodyReader *bytes.Reader
			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewReader(bodyBytes)
			} else {
				bodyReader = bytes.NewReader([]byte{})
			}

			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.NotEmpty(t, w.Header().Get("X-Request-ID"), "All routes should have Request ID")
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestNewRouter_PathMatching(t *testing.T) {
	// Test that path suffix matching works correctly
	tests := []struct {
		name           string
		path           string
		method         string
		shouldMatch    bool
		expectedStatus int
	}{
		{
			name:           "versions suffix match",
			path:           "/schemas/test-id/versions",
			method:         http.MethodGet,
			shouldMatch:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "disable suffix match",
			path:           "/schemas/test-id/disable",
			method:         http.MethodPut,
			shouldMatch:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "enable suffix match",
			path:           "/schemas/test-id/enable",
			method:         http.MethodPut,
			shouldMatch:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "parameters suffix match",
			path:           "/schemas/test-id/parameters",
			method:         http.MethodPut,
			shouldMatch:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "schema ID without suffix - GET",
			path:           "/schemas/test-id",
			method:         http.MethodGet,
			shouldMatch:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "schema ID without suffix - PUT",
			path:           "/schemas/test-id",
			method:         http.MethodPut,
			shouldMatch:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "schema ID without suffix - DELETE",
			path:           "/schemas/test-id",
			method:         http.MethodDelete,
			shouldMatch:    true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := service.NewService(mockRepo)
			handler := handlers.NewHandler(svc)
			router := NewRouter(handler)

			// Setup appropriate mocks based on path
			schema := createTestSchema()
			if strings.HasSuffix(tt.path, "/versions") {
				versions := []*models.DetectionSchemaVersion{
					{
						VersionID: "v1",
						Version:   1,
						Title:     "Test Rule",
						CreatedBy: "test-user",
						CreatedAt: time.Now(),
					},
				}
				mockRepo.On("GetSchemaVersionHistory", mock.Anything, mock.Anything).Return(versions, nil)
			} else if strings.HasSuffix(tt.path, "/disable") {
				mockRepo.On("GetSchemaByVersionID", mock.Anything, mock.Anything).Return(schema, nil)
				mockRepo.On("DisableSchema", mock.Anything, mock.Anything, "00000000-0000-0000-0000-000000000001").Return(nil)
			} else if strings.HasSuffix(tt.path, "/enable") {
				mockRepo.On("EnableSchema", mock.Anything, mock.Anything).Return(nil)
			} else if strings.HasSuffix(tt.path, "/parameters") {
				mockRepo.On("SetActiveParameterSet", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			} else {
				// Schema GET/PUT/DELETE
				if tt.method == http.MethodGet {
					mockRepo.On("GetSchemaByVersionID", mock.Anything, mock.Anything).Return(schema, nil)
				} else if tt.method == http.MethodPut {
					mockRepo.On("GetLatestSchemaByID", mock.Anything, mock.Anything).Return(schema, nil)
					mockRepo.On("CreateSchema", mock.Anything, mock.Anything).Return(nil)
					mockRepo.On("GetSchemaByVersionID", mock.Anything, mock.Anything).Return(schema, nil)
				} else if tt.method == http.MethodDelete {
					mockRepo.On("GetSchemaByVersionID", mock.Anything, mock.Anything).Return(schema, nil)
					mockRepo.On("HideSchema", mock.Anything, mock.Anything, "00000000-0000-0000-0000-000000000001").Return(nil)
				}
			}

			var body *bytes.Reader
			if tt.method == http.MethodPut && strings.HasSuffix(tt.path, "/parameters") {
				bodyBytes, _ := json.Marshal(map[string]interface{}{"active_parameter_set": "default"})
				body = bytes.NewReader(bodyBytes)
			} else if tt.method == http.MethodPut {
				bodyBytes, _ := json.Marshal(map[string]interface{}{
					"model":      schema.Model,
					"view":       schema.View,
					"controller": schema.Controller,
				})
				body = bytes.NewReader(bodyBytes)
			} else {
				body = bytes.NewReader([]byte{})
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.method != http.MethodGet {
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-User-ID", "test-user")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}
