package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/models"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/service"
)

// MockRepository is a mock implementation of repository.Repository
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

func (m *MockRepository) Close() {
	m.Called()
}

// Helper function to create test schema
func createTestDetectionSchema() *models.DetectionSchema {
	return &models.DetectionSchema{
		ID:        uuid.NewString(),
		VersionID: uuid.NewString(),
		Model: map[string]interface{}{
			"aggregation": "count",
			"threshold":   10,
		},
		View: map[string]interface{}{
			"title":    "Test Rule",
			"severity": "high",
		},
		Controller: map[string]interface{}{
			"query": "test",
		},
		CreatedBy: uuid.NewString(),
		Version:   1,
	}
}

func TestHealthCheck(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	handler.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestGetCorrelationTypes(t *testing.T) {
	handler := &Handler{}

	t.Run("successful GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/correlation/types", nil)
		w := httptest.NewRecorder()

		handler.GetCorrelationTypes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].([]interface{})
		assert.True(t, ok)
		assert.NotEmpty(t, data)
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/correlation/types", nil)
		w := httptest.NewRecorder()

		handler.GetCorrelationTypes(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestCreateSchema(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		createReq := &models.CreateSchemaRequest{
			Model:      map[string]interface{}{"threshold": 10},
			View:       map[string]interface{}{"title": "Test"},
			Controller: map[string]interface{}{"query": "test"},
		}

		schema := createTestDetectionSchema()

		mockRepo.On("CreateSchema", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("GetSchemaByVersionID", mock.Anything, mock.Anything).Return(schema, nil)

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.CreateSchema(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		// Content-Type is set by httputil.WriteJSON
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas", strings.NewReader("invalid"))
		w := httptest.NewRecorder()

		handler.CreateSchema(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
		w := httptest.NewRecorder()

		handler.CreateSchema(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestUpdateSchema(t *testing.T) {
	schemaID := uuid.NewString()

	t.Run("successful update", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		updateReq := &models.UpdateSchemaRequest{
			Model:      map[string]interface{}{"threshold": 20},
			View:       map[string]interface{}{"title": "Updated"},
			Controller: map[string]interface{}{"query": "updated"},
		}

		existing := createTestDetectionSchema()
		existing.ID = schemaID

		schema := createTestDetectionSchema()
		schema.ID = schemaID

		mockRepo.On("GetLatestSchemaByID", mock.Anything, schemaID).Return(existing, nil)
		mockRepo.On("CreateSchema", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("GetSchemaByVersionID", mock.Anything, mock.Anything).Return(schema, nil)

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest(http.MethodPut, "/schemas/"+schemaID, bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.UpdateSchema(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("builtin rule protection", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		updateReq := &models.UpdateSchemaRequest{
			Model:      map[string]interface{}{"threshold": 20},
			View:       map[string]interface{}{"title": "Updated"},
			Controller: map[string]interface{}{"query": "updated"},
		}

		builtinSchema := createTestDetectionSchema()
		builtinSchema.Controller = map[string]interface{}{
			"query": "test",
			"metadata": map[string]interface{}{
				"source": "builtin",
			},
		}

		mockRepo.On("GetLatestSchemaByID", mock.Anything, schemaID).Return(builtinSchema, nil)

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest(http.MethodPut, "/schemas/"+schemaID, bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.UpdateSchema(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "/schemas/"+schemaID, nil)
		w := httptest.NewRecorder()

		handler.UpdateSchema(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestListSchemas(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		schemas := []*models.DetectionSchema{createTestDetectionSchema(), createTestDetectionSchema()}

		mockRepo.On("ListSchemas", mock.Anything, mock.Anything).Return(schemas, 2, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
		w := httptest.NewRecorder()

		handler.ListSchemas(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("with query parameters", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		mockRepo.On("ListSchemas", mock.Anything, mock.MatchedBy(func(req *models.ListSchemasRequest) bool {
			return req.Severity == "high" && req.Page == 2
		})).Return([]*models.DetectionSchema{}, 0, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas?page=2&severity=high", nil)
		w := httptest.NewRecorder()

		handler.ListSchemas(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas", nil)
		w := httptest.NewRecorder()

		handler.ListSchemas(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestGetSchema(t *testing.T) {
	schemaID := uuid.NewString()

	t.Run("successful get", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		schema := createTestDetectionSchema()
		schema.ID = schemaID

		mockRepo.On("GetSchemaByVersionID", mock.Anything, schemaID).Return(schema, nil)

		req := httptest.NewRequest(http.MethodGet, "/schemas/"+schemaID, nil)
		w := httptest.NewRecorder()

		handler.GetSchema(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("schema not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		mockRepo.On("GetSchemaByVersionID", mock.Anything, schemaID).Return(nil, repository.ErrSchemaNotFound)
		mockRepo.On("GetLatestSchemaByID", mock.Anything, schemaID).Return(nil, repository.ErrSchemaNotFound)

		req := httptest.NewRequest(http.MethodGet, "/schemas/"+schemaID, nil)
		w := httptest.NewRecorder()

		handler.GetSchema(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodPost, "/schemas/"+schemaID, nil)
		w := httptest.NewRecorder()

		handler.GetSchema(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestGetVersionHistory(t *testing.T) {
	schemaID := uuid.NewString()

	t.Run("successful get", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		versions := []*models.DetectionSchemaVersion{
			{VersionID: uuid.NewString(), Version: 2, Title: "V2"},
			{VersionID: uuid.NewString(), Version: 1, Title: "V1"},
		}

		mockRepo.On("GetSchemaVersionHistory", mock.Anything, schemaID).Return(versions, nil)

		req := httptest.NewRequest(http.MethodGet, "/schemas/"+schemaID+"/versions", nil)
		w := httptest.NewRecorder()

		handler.GetVersionHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("schema not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		mockRepo.On("GetSchemaVersionHistory", mock.Anything, schemaID).Return(nil, repository.ErrSchemaNotFound)

		req := httptest.NewRequest(http.MethodGet, "/schemas/"+schemaID+"/versions", nil)
		w := httptest.NewRecorder()

		handler.GetVersionHistory(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodPost, "/schemas/"+schemaID+"/versions", nil)
		w := httptest.NewRecorder()

		handler.GetVersionHistory(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestDisableSchema(t *testing.T) {
	versionID := uuid.NewString()

	t.Run("successful disable", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		schema := createTestDetectionSchema()

		mockRepo.On("GetSchemaByVersionID", mock.Anything, versionID).Return(schema, nil)
		mockRepo.On("DisableSchema", mock.Anything, versionID, mock.Anything).Return(nil)

		req := httptest.NewRequest(http.MethodPut, "/schemas/"+versionID+"/disable", nil)
		w := httptest.NewRecorder()

		handler.DisableSchema(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("builtin rule protection", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		builtinSchema := createTestDetectionSchema()
		builtinSchema.Controller = map[string]interface{}{
			"metadata": map[string]interface{}{
				"source": "builtin",
			},
		}

		mockRepo.On("GetSchemaByVersionID", mock.Anything, versionID).Return(builtinSchema, nil)

		req := httptest.NewRequest(http.MethodPut, "/schemas/"+versionID+"/disable", nil)
		w := httptest.NewRecorder()

		handler.DisableSchema(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "/schemas/"+versionID+"/disable", nil)
		w := httptest.NewRecorder()

		handler.DisableSchema(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestEnableSchema(t *testing.T) {
	versionID := uuid.NewString()

	t.Run("successful enable", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		mockRepo.On("EnableSchema", mock.Anything, versionID).Return(nil)

		req := httptest.NewRequest(http.MethodPut, "/schemas/"+versionID+"/enable", nil)
		w := httptest.NewRecorder()

		handler.EnableSchema(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		mockRepo.On("EnableSchema", mock.Anything, versionID).Return(errors.New("db error"))

		req := httptest.NewRequest(http.MethodPut, "/schemas/"+versionID+"/enable", nil)
		w := httptest.NewRecorder()

		handler.EnableSchema(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "/schemas/"+versionID+"/enable", nil)
		w := httptest.NewRecorder()

		handler.EnableSchema(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestHideSchema(t *testing.T) {
	versionID := uuid.NewString()

	t.Run("successful hide", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		schema := createTestDetectionSchema()

		mockRepo.On("GetSchemaByVersionID", mock.Anything, versionID).Return(schema, nil)
		mockRepo.On("HideSchema", mock.Anything, versionID, mock.Anything).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/schemas/"+versionID, nil)
		w := httptest.NewRecorder()

		handler.HideSchema(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("builtin rule protection", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		builtinSchema := createTestDetectionSchema()
		builtinSchema.Controller = map[string]interface{}{
			"metadata": map[string]interface{}{
				"source": "builtin",
			},
		}

		mockRepo.On("GetSchemaByVersionID", mock.Anything, versionID).Return(builtinSchema, nil)

		req := httptest.NewRequest(http.MethodDelete, "/schemas/"+versionID, nil)
		w := httptest.NewRecorder()

		handler.HideSchema(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "/schemas/"+versionID, nil)
		w := httptest.NewRecorder()

		handler.HideSchema(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestSetActiveParameterSet(t *testing.T) {
	versionID := uuid.NewString()

	t.Run("successful set", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.NewService(mockRepo)
		handler := NewHandler(svc)

		mockRepo.On("SetActiveParameterSet", mock.Anything, versionID, "aggressive").Return(nil)

		reqBody := map[string]string{"active_parameter_set": "aggressive"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/schemas/"+versionID+"/parameters", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.SetActiveParameterSet(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodPut, "/schemas/"+versionID+"/parameters", strings.NewReader("invalid"))
		w := httptest.NewRecorder()

		handler.SetActiveParameterSet(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing parameter", func(t *testing.T) {
		handler := &Handler{}
		reqBody := map[string]string{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/schemas/"+versionID+"/parameters", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.SetActiveParameterSet(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("wrong method", func(t *testing.T) {
		handler := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "/schemas/"+versionID+"/parameters", nil)
		w := httptest.NewRecorder()

		handler.SetActiveParameterSet(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal int
		expected   int
	}{
		{"valid integer", "42", 10, 42},
		{"empty string", "", 10, 10},
		{"invalid string", "abc", 10, 10},
		{"negative number", "-5", 10, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt(tt.input, tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToJSONAPIResource(t *testing.T) {
	schema := createTestDetectionSchema()
	result := toJSONAPIResource(schema)

	assert.Equal(t, "detection-schema", result["type"])
	assert.Equal(t, schema.ID, result["id"])

	attrs, ok := result["attributes"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, schema.VersionID, attrs["version_id"])
}

func TestToJSONAPICollection(t *testing.T) {
	schemas := []*models.DetectionSchema{createTestDetectionSchema()}
	pagination := &models.Pagination{Page: 1, Limit: 50, Total: 1, TotalPages: 1}

	result := toJSONAPICollection(schemas, pagination)

	data, ok := result["data"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, data, 1)

	meta, ok := result["meta"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, meta, "pagination")
}

func TestToJSONAPIVersionCollection(t *testing.T) {
	versions := []*models.DetectionSchemaVersion{
		{VersionID: uuid.NewString(), Version: 2, Title: "V2"},
		{VersionID: uuid.NewString(), Version: 1, Title: "V1"},
	}

	result := toJSONAPIVersionCollection(versions)

	data, ok := result["data"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, data, 2)

	firstVersion := data[0]
	assert.Equal(t, "detection-schema-version", firstVersion["type"])
}
