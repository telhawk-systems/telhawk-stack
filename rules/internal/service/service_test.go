package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/models"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/repository"
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

// Helper functions
func createTestRequest() *models.CreateSchemaRequest {
	return &models.CreateSchemaRequest{
		Model: map[string]interface{}{
			"aggregation": "count",
			"threshold":   10,
		},
		View: map[string]interface{}{
			"title":       "Test Detection Rule",
			"severity":    "high",
			"description": "Test description",
		},
		Controller: map[string]interface{}{
			"query":    "severity:high",
			"interval": "5m",
		},
	}
}

func createBuiltinSchema(id, versionID string) *models.DetectionSchema {
	return &models.DetectionSchema{
		ID:        id,
		VersionID: versionID,
		Model: map[string]interface{}{
			"aggregation": "count",
		},
		View: map[string]interface{}{
			"title": "Builtin Rule",
		},
		Controller: map[string]interface{}{
			"query": "test",
			"metadata": map[string]interface{}{
				"source": "builtin",
			},
		},
		CreatedBy: "system",
	}
}

func createUserSchema(id, versionID, userID string) *models.DetectionSchema {
	return &models.DetectionSchema{
		ID:        id,
		VersionID: versionID,
		Model: map[string]interface{}{
			"aggregation": "count",
		},
		View: map[string]interface{}{
			"title": "User Rule",
		},
		Controller: map[string]interface{}{
			"query": "test",
		},
		CreatedBy: userID,
	}
}

func TestNewService(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
}

func TestCreateSchema(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("successful creation", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		req := createTestRequest()

		createdSchema := &models.DetectionSchema{
			ID:         uuid.NewString(),
			VersionID:  uuid.NewString(),
			Model:      req.Model,
			View:       req.View,
			Controller: req.Controller,
			CreatedBy:  userID,
			Version:    1,
		}

		// Expect CreateSchema to be called
		mockRepo.On("CreateSchema", ctx, mock.MatchedBy(func(s *models.DetectionSchema) bool {
			// Verify UUIDs were generated
			return s.ID != "" && s.VersionID != "" &&
				s.CreatedBy == userID &&
				s.Model != nil && s.View != nil && s.Controller != nil
		})).Return(nil)

		// Expect GetSchemaByVersionID to be called to retrieve with version number
		mockRepo.On("GetSchemaByVersionID", ctx, mock.Anything).Return(createdSchema, nil)

		result, err := service.CreateSchema(ctx, req, userID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, createdSchema.ID, result.ID)
		assert.Equal(t, 1, result.Version)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		req := createTestRequest()

		expectedErr := errors.New("database error")
		mockRepo.On("CreateSchema", ctx, mock.Anything).Return(expectedErr)

		result, err := service.CreateSchema(ctx, req, userID)

		assert.ErrorIs(t, err, expectedErr)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateSchema(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewString()
	stableID := uuid.NewString()

	t.Run("successful update", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		req := &models.UpdateSchemaRequest{
			Model:      map[string]interface{}{"threshold": 20},
			View:       map[string]interface{}{"title": "Updated Rule"},
			Controller: map[string]interface{}{"query": "updated"},
		}

		existingSchema := createUserSchema(stableID, uuid.NewString(), userID)
		updatedSchema := &models.DetectionSchema{
			ID:         stableID,
			VersionID:  uuid.NewString(),
			Model:      req.Model,
			View:       req.View,
			Controller: req.Controller,
			CreatedBy:  userID,
			Version:    2,
		}

		mockRepo.On("GetLatestSchemaByID", ctx, stableID).Return(existingSchema, nil)
		mockRepo.On("CreateSchema", ctx, mock.MatchedBy(func(s *models.DetectionSchema) bool {
			return s.ID == stableID && s.VersionID != "" && s.CreatedBy == userID
		})).Return(nil)
		mockRepo.On("GetSchemaByVersionID", ctx, mock.Anything).Return(updatedSchema, nil)

		result, err := service.UpdateSchema(ctx, stableID, req, userID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, stableID, result.ID)
		assert.Equal(t, 2, result.Version)
		mockRepo.AssertExpectations(t)
	})

	t.Run("rule not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		req := &models.UpdateSchemaRequest{
			Model:      map[string]interface{}{"threshold": 20},
			View:       map[string]interface{}{"title": "Updated Rule"},
			Controller: map[string]interface{}{"query": "updated"},
		}

		mockRepo.On("GetLatestSchemaByID", ctx, stableID).Return(nil, repository.ErrSchemaNotFound)

		result, err := service.UpdateSchema(ctx, stableID, req, userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "rule not found")
		mockRepo.AssertExpectations(t)
	})

	t.Run("builtin rule protection", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		req := &models.UpdateSchemaRequest{
			Model:      map[string]interface{}{"threshold": 20},
			View:       map[string]interface{}{"title": "Updated Rule"},
			Controller: map[string]interface{}{"query": "updated"},
		}

		builtinSchema := createBuiltinSchema(stableID, uuid.NewString())

		mockRepo.On("GetLatestSchemaByID", ctx, stableID).Return(builtinSchema, nil)

		result, err := service.UpdateSchema(ctx, stableID, req, userID)

		assert.ErrorIs(t, err, ErrBuiltinRuleProtected)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetSchema(t *testing.T) {
	ctx := context.Background()
	versionID := uuid.NewString()
	stableID := uuid.NewString()

	t.Run("get by version ID", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		schema := createUserSchema(stableID, versionID, uuid.NewString())

		mockRepo.On("GetSchemaByVersionID", ctx, versionID).Return(schema, nil)

		result, err := service.GetSchema(ctx, versionID, nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, versionID, result.VersionID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("get by stable ID (latest)", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		schema := createUserSchema(stableID, versionID, uuid.NewString())

		// First try as version ID (fails)
		mockRepo.On("GetSchemaByVersionID", ctx, stableID).Return(nil, repository.ErrSchemaNotFound)
		// Then try as stable ID (succeeds)
		mockRepo.On("GetLatestSchemaByID", ctx, stableID).Return(schema, nil)

		result, err := service.GetSchema(ctx, stableID, nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, stableID, result.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("schema not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		nonexistentID := uuid.NewString()

		mockRepo.On("GetSchemaByVersionID", ctx, nonexistentID).Return(nil, repository.ErrSchemaNotFound)
		mockRepo.On("GetLatestSchemaByID", ctx, nonexistentID).Return(nil, repository.ErrSchemaNotFound)

		result, err := service.GetSchema(ctx, nonexistentID, nil)

		assert.ErrorIs(t, err, repository.ErrSchemaNotFound)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		dbError := errors.New("database error")
		mockRepo.On("GetSchemaByVersionID", ctx, versionID).Return(nil, dbError)

		result, err := service.GetSchema(ctx, versionID, nil)

		assert.ErrorIs(t, err, dbError)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestListSchemas(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list with defaults", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		schemas := []*models.DetectionSchema{
			createUserSchema(uuid.NewString(), uuid.NewString(), uuid.NewString()),
			createUserSchema(uuid.NewString(), uuid.NewString(), uuid.NewString()),
		}

		req := &models.ListSchemasRequest{
			Page:  0, // Invalid, should default to 1
			Limit: 0, // Invalid, should default to 50
		}

		mockRepo.On("ListSchemas", ctx, mock.MatchedBy(func(r *models.ListSchemasRequest) bool {
			return r.Page == 1 && r.Limit == 50
		})).Return(schemas, 2, nil)

		result, err := service.ListSchemas(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Schemas, 2)
		assert.Equal(t, 1, result.Pagination.Page)
		assert.Equal(t, 50, result.Pagination.Limit)
		assert.Equal(t, 2, result.Pagination.Total)
		assert.Equal(t, 1, result.Pagination.TotalPages)
		mockRepo.AssertExpectations(t)
	})

	t.Run("pagination calculation", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		req := &models.ListSchemasRequest{
			Page:  2,
			Limit: 10,
		}

		mockRepo.On("ListSchemas", ctx, req).Return([]*models.DetectionSchema{}, 25, nil)

		result, err := service.ListSchemas(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 25, result.Pagination.Total)
		assert.Equal(t, 3, result.Pagination.TotalPages) // 25 items / 10 per page = 3 pages
		mockRepo.AssertExpectations(t)
	})

	t.Run("limit exceeds maximum", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		req := &models.ListSchemasRequest{
			Page:  1,
			Limit: 200, // Exceeds max of 100
		}

		mockRepo.On("ListSchemas", ctx, mock.MatchedBy(func(r *models.ListSchemasRequest) bool {
			return r.Limit == 50 // Should be capped to default
		})).Return([]*models.DetectionSchema{}, 0, nil)

		result, err := service.ListSchemas(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 50, result.Pagination.Limit)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		req := &models.ListSchemasRequest{
			Page:  1,
			Limit: 10,
		}

		dbError := errors.New("database error")
		mockRepo.On("ListSchemas", ctx, req).Return(nil, 0, dbError)

		result, err := service.ListSchemas(ctx, req)

		assert.ErrorIs(t, err, dbError)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetVersionHistory(t *testing.T) {
	ctx := context.Background()
	stableID := uuid.NewString()

	t.Run("successful version history", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		versions := []*models.DetectionSchemaVersion{
			{VersionID: uuid.NewString(), Version: 3, Title: "Version 3"},
			{VersionID: uuid.NewString(), Version: 2, Title: "Version 2"},
			{VersionID: uuid.NewString(), Version: 1, Title: "Version 1"},
		}

		mockRepo.On("GetSchemaVersionHistory", ctx, stableID).Return(versions, nil)

		result, err := service.GetVersionHistory(ctx, stableID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, stableID, result.ID)
		assert.Equal(t, "Version 3", result.Title) // Latest version title
		assert.Len(t, result.Versions, 3)
		mockRepo.AssertExpectations(t)
	})

	t.Run("schema not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("GetSchemaVersionHistory", ctx, stableID).Return(nil, repository.ErrSchemaNotFound)

		result, err := service.GetVersionHistory(ctx, stableID)

		assert.ErrorIs(t, err, repository.ErrSchemaNotFound)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty version list", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("GetSchemaVersionHistory", ctx, stableID).Return([]*models.DetectionSchemaVersion{}, nil)

		result, err := service.GetVersionHistory(ctx, stableID)

		assert.ErrorIs(t, err, repository.ErrSchemaNotFound)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

func TestDisableSchema(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewString()
	versionID := uuid.NewString()

	t.Run("disable user schema", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		schema := createUserSchema(uuid.NewString(), versionID, userID)

		mockRepo.On("GetSchemaByVersionID", ctx, versionID).Return(schema, nil)
		mockRepo.On("DisableSchema", ctx, versionID, userID).Return(nil)

		err := service.DisableSchema(ctx, versionID, userID)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("disable builtin schema", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		schema := createBuiltinSchema(uuid.NewString(), versionID)

		mockRepo.On("GetSchemaByVersionID", ctx, versionID).Return(schema, nil)

		err := service.DisableSchema(ctx, versionID, userID)

		assert.ErrorIs(t, err, ErrBuiltinRuleProtected)
		mockRepo.AssertExpectations(t)
	})

	t.Run("schema not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("GetSchemaByVersionID", ctx, versionID).Return(nil, repository.ErrSchemaNotFound)

		err := service.DisableSchema(ctx, versionID, userID)

		assert.ErrorIs(t, err, repository.ErrSchemaNotFound)
		mockRepo.AssertExpectations(t)
	})
}

func TestEnableSchema(t *testing.T) {
	ctx := context.Background()
	versionID := uuid.NewString()

	t.Run("enable schema", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("EnableSchema", ctx, versionID).Return(nil)

		err := service.EnableSchema(ctx, versionID)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		dbError := errors.New("database error")
		mockRepo.On("EnableSchema", ctx, versionID).Return(dbError)

		err := service.EnableSchema(ctx, versionID)

		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}

func TestHideSchema(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewString()
	versionID := uuid.NewString()

	t.Run("hide user schema", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		schema := createUserSchema(uuid.NewString(), versionID, userID)

		mockRepo.On("GetSchemaByVersionID", ctx, versionID).Return(schema, nil)
		mockRepo.On("HideSchema", ctx, versionID, userID).Return(nil)

		err := service.HideSchema(ctx, versionID, userID)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("hide builtin schema", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		schema := createBuiltinSchema(uuid.NewString(), versionID)

		mockRepo.On("GetSchemaByVersionID", ctx, versionID).Return(schema, nil)

		err := service.HideSchema(ctx, versionID, userID)

		assert.ErrorIs(t, err, ErrBuiltinRuleProtected)
		mockRepo.AssertExpectations(t)
	})

	t.Run("schema not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("GetSchemaByVersionID", ctx, versionID).Return(nil, repository.ErrSchemaNotFound)

		err := service.HideSchema(ctx, versionID, userID)

		assert.ErrorIs(t, err, repository.ErrSchemaNotFound)
		mockRepo.AssertExpectations(t)
	})
}

func TestSetActiveParameterSet(t *testing.T) {
	ctx := context.Background()
	versionID := uuid.NewString()

	t.Run("set parameter set", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		mockRepo.On("SetActiveParameterSet", ctx, versionID, "aggressive").Return(nil)

		err := service.SetActiveParameterSet(ctx, versionID, "aggressive")

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)

		dbError := errors.New("database error")
		mockRepo.On("SetActiveParameterSet", ctx, versionID, "balanced").Return(dbError)

		err := service.SetActiveParameterSet(ctx, versionID, "balanced")

		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}

func TestIsBuiltinRule(t *testing.T) {
	service := NewService(nil)

	tests := []struct {
		name     string
		schema   *models.DetectionSchema
		expected bool
	}{
		{
			name:     "builtin rule",
			schema:   createBuiltinSchema(uuid.NewString(), uuid.NewString()),
			expected: true,
		},
		{
			name:     "user rule",
			schema:   createUserSchema(uuid.NewString(), uuid.NewString(), uuid.NewString()),
			expected: false,
		},
		{
			name: "nil controller",
			schema: &models.DetectionSchema{
				Controller: nil,
			},
			expected: false,
		},
		{
			name: "no metadata",
			schema: &models.DetectionSchema{
				Controller: map[string]interface{}{
					"query": "test",
				},
			},
			expected: false,
		},
		{
			name: "metadata not a map",
			schema: &models.DetectionSchema{
				Controller: map[string]interface{}{
					"metadata": "string",
				},
			},
			expected: false,
		},
		{
			name: "no source in metadata",
			schema: &models.DetectionSchema{
				Controller: map[string]interface{}{
					"metadata": map[string]interface{}{
						"other": "value",
					},
				},
			},
			expected: false,
		},
		{
			name: "source not a string",
			schema: &models.DetectionSchema{
				Controller: map[string]interface{}{
					"metadata": map[string]interface{}{
						"source": 123,
					},
				},
			},
			expected: false,
		},
		{
			name: "source not builtin",
			schema: &models.DetectionSchema{
				Controller: map[string]interface{}{
					"metadata": map[string]interface{}{
						"source": "user",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isBuiltinRule(tt.schema)
			assert.Equal(t, tt.expected, result)
		})
	}
}
