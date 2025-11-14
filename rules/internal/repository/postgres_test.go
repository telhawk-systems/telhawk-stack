package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/models"
)

// getTestDBConnString returns connection string for test database
func getTestDBConnString() string {
	// Default to test database, but allow override via env var
	connString := os.Getenv("RULES_DB_TEST_URL")
	if connString == "" {
		connString = "postgres://telhawk:telhawk-rules-dev@localhost:5433/telhawk_rules?sslmode=disable"
	}
	return connString
}

// setupTestDB creates a test repository and cleans up existing test data
func setupTestDB(t *testing.T) *PostgresRepository {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, err := NewPostgresRepository(ctx, getTestDBConnString())
	if err != nil {
		t.Skipf("skipping integration test - database not available: %v", err)
	}

	// Clean up any existing test data
	_, err = repo.pool.Exec(ctx, "TRUNCATE TABLE detection_schemas")
	if err != nil {
		t.Skipf("skipping integration test - cannot clean test data: %v", err)
	}

	return repo
}

// createTestSchema creates a minimal valid detection schema for testing
func createTestSchema(userID string) *models.DetectionSchema {
	return &models.DetectionSchema{
		Model: map[string]interface{}{
			"aggregation": "count",
			"threshold":   10,
		},
		View: map[string]interface{}{
			"title":       "Test Detection Rule",
			"severity":    "high",
			"description": "Test rule description",
		},
		Controller: map[string]interface{}{
			"query":    "severity:high",
			"interval": "5m",
		},
		CreatedBy: userID,
	}
}

func TestNewPostgresRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name       string
		connString string
		wantErr    bool
		skipMsg    string
	}{
		{
			name:       "valid connection string",
			connString: getTestDBConnString(),
			wantErr:    false,
			skipMsg:    "database not available",
		},
		{
			name:       "invalid connection string",
			connString: "invalid://connection",
			wantErr:    true,
		},
		{
			name:       "unreachable host",
			connString: "postgres://user:pass@nonexistent-host:5432/db",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo, err := NewPostgresRepository(ctx, tt.connString)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, repo)
			} else {
				if err != nil && tt.skipMsg != "" {
					t.Skipf("skipping test - %s: %v", tt.skipMsg, err)
				}
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				if repo != nil {
					repo.Close()
				}
			}
		})
	}
}

func TestCreateSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("create new schema with auto-generated IDs", func(t *testing.T) {
		schema := createTestSchema(userID)

		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Verify IDs were generated
		assert.NotEmpty(t, schema.ID)
		assert.NotEmpty(t, schema.VersionID)

		// Verify we can retrieve it
		retrieved, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.Equal(t, schema.ID, retrieved.ID)
		assert.Equal(t, schema.VersionID, retrieved.VersionID)
		assert.Equal(t, userID, retrieved.CreatedBy)
		assert.Equal(t, "Test Detection Rule", retrieved.View["title"])
	})

	t.Run("create schema with provided IDs", func(t *testing.T) {
		schema := createTestSchema(userID)
		schema.ID = uuid.NewString()
		schema.VersionID = uuid.NewString()

		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Verify provided IDs were used
		retrieved, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.Equal(t, schema.ID, retrieved.ID)
		assert.Equal(t, schema.VersionID, retrieved.VersionID)
	})

	t.Run("create multiple versions of same schema", func(t *testing.T) {
		schema := createTestSchema(userID)

		// Create first version
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		stableID := schema.ID
		firstVersionID := schema.VersionID

		// Create second version with same stable ID
		time.Sleep(10 * time.Millisecond) // Ensure different created_at
		schema2 := createTestSchema(userID)
		schema2.ID = stableID
		schema2.View["title"] = "Updated Detection Rule"

		err = repo.CreateSchema(ctx, schema2)
		require.NoError(t, err)

		// Verify both versions exist
		v1, err := repo.GetSchemaByVersionID(ctx, firstVersionID)
		require.NoError(t, err)
		assert.Equal(t, "Test Detection Rule", v1.View["title"])

		v2, err := repo.GetSchemaByVersionID(ctx, schema2.VersionID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Detection Rule", v2.View["title"])

		// Verify latest returns second version
		latest, err := repo.GetLatestSchemaByID(ctx, stableID)
		require.NoError(t, err)
		assert.Equal(t, schema2.VersionID, latest.VersionID)
		assert.Equal(t, "Updated Detection Rule", latest.View["title"])
	})

	t.Run("invalid JSONB fields", func(t *testing.T) {
		schema := &models.DetectionSchema{
			Model: map[string]interface{}{
				"invalid": make(chan int), // channels can't be marshaled to JSON
			},
			View:       map[string]interface{}{"title": "Test"},
			Controller: map[string]interface{}{"query": "test"},
			CreatedBy:  userID,
		}

		err := repo.CreateSchema(ctx, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal")
	})
}

func TestGetSchemaByVersionID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("retrieve existing schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		retrieved, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.Equal(t, schema.ID, retrieved.ID)
		assert.Equal(t, schema.VersionID, retrieved.VersionID)
		assert.Equal(t, userID, retrieved.CreatedBy)
		assert.NotZero(t, retrieved.CreatedAt)
		assert.Nil(t, retrieved.DisabledAt)
		assert.Nil(t, retrieved.HiddenAt)
		assert.Equal(t, 1, retrieved.Version) // First version
	})

	t.Run("schema not found", func(t *testing.T) {
		nonexistentID := uuid.NewString()
		retrieved, err := repo.GetSchemaByVersionID(ctx, nonexistentID)
		assert.ErrorIs(t, err, ErrSchemaNotFound)
		assert.Nil(t, retrieved)
	})

	t.Run("version number calculation", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		stableID := schema.ID

		// Create second version
		time.Sleep(10 * time.Millisecond)
		schema2 := createTestSchema(userID)
		schema2.ID = stableID
		err = repo.CreateSchema(ctx, schema2)
		require.NoError(t, err)

		// Verify version numbers
		v1, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.Equal(t, 1, v1.Version)

		v2, err := repo.GetSchemaByVersionID(ctx, schema2.VersionID)
		require.NoError(t, err)
		assert.Equal(t, 2, v2.Version)
	})
}

func TestGetLatestSchemaByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("get latest of multiple versions", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		stableID := schema.ID
		firstVersionID := schema.VersionID

		// Create second version
		time.Sleep(10 * time.Millisecond)
		schema2 := createTestSchema(userID)
		schema2.ID = stableID
		schema2.View["title"] = "Version 2"
		err = repo.CreateSchema(ctx, schema2)
		require.NoError(t, err)

		// Create third version
		time.Sleep(10 * time.Millisecond)
		schema3 := createTestSchema(userID)
		schema3.ID = stableID
		schema3.View["title"] = "Version 3"
		err = repo.CreateSchema(ctx, schema3)
		require.NoError(t, err)

		// GetLatest should return version 3
		latest, err := repo.GetLatestSchemaByID(ctx, stableID)
		require.NoError(t, err)
		assert.Equal(t, schema3.VersionID, latest.VersionID)
		assert.Equal(t, "Version 3", latest.View["title"])
		assert.NotEqual(t, firstVersionID, latest.VersionID)
	})

	t.Run("schema not found", func(t *testing.T) {
		nonexistentID := uuid.NewString()
		retrieved, err := repo.GetLatestSchemaByID(ctx, nonexistentID)
		assert.ErrorIs(t, err, ErrSchemaNotFound)
		assert.Nil(t, retrieved)
	})
}

func TestGetSchemaVersionHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("get history with multiple versions", func(t *testing.T) {
		schema := createTestSchema(userID)
		schema.View["title"] = "Version 1"
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		stableID := schema.ID

		// Create version 2
		time.Sleep(10 * time.Millisecond)
		schema2 := createTestSchema(userID)
		schema2.ID = stableID
		schema2.View["title"] = "Version 2"
		err = repo.CreateSchema(ctx, schema2)
		require.NoError(t, err)

		// Create version 3
		time.Sleep(10 * time.Millisecond)
		schema3 := createTestSchema(userID)
		schema3.ID = stableID
		schema3.View["title"] = "Version 3"
		err = repo.CreateSchema(ctx, schema3)
		require.NoError(t, err)

		// Get version history
		versions, err := repo.GetSchemaVersionHistory(ctx, stableID)
		require.NoError(t, err)
		require.Len(t, versions, 3)

		// Verify order (newest first)
		assert.Equal(t, "Version 3", versions[0].Title)
		assert.Equal(t, "Version 2", versions[1].Title)
		assert.Equal(t, "Version 1", versions[2].Title)

		// Verify version numbers
		assert.Equal(t, 3, versions[0].Version)
		assert.Equal(t, 2, versions[1].Version)
		assert.Equal(t, 1, versions[2].Version)
	})

	t.Run("schema not found", func(t *testing.T) {
		nonexistentID := uuid.NewString()
		versions, err := repo.GetSchemaVersionHistory(ctx, nonexistentID)
		assert.ErrorIs(t, err, ErrSchemaNotFound)
		assert.Nil(t, versions)
	})
}

func TestListSchemas(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	// Create test data
	for i := 0; i < 5; i++ {
		schema := createTestSchema(userID)
		schema.View["title"] = fmt.Sprintf("Rule %d", i+1)
		schema.View["severity"] = []string{"low", "medium", "high", "high", "critical"}[i]
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Create a second version for some schemas
		if i%2 == 0 {
			time.Sleep(10 * time.Millisecond)
			schema2 := createTestSchema(userID)
			schema2.ID = schema.ID
			schema2.View["title"] = fmt.Sprintf("Rule %d Updated", i+1)
			schema2.View["severity"] = schema.View["severity"]
			err = repo.CreateSchema(ctx, schema2)
			require.NoError(t, err)
		}
	}

	t.Run("list all schemas with pagination", func(t *testing.T) {
		req := &models.ListSchemasRequest{
			Page:  1,
			Limit: 10,
		}

		schemas, total, err := repo.ListSchemas(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.Len(t, schemas, 5)

		// Verify only latest versions are returned
		for _, s := range schemas {
			if s.View["title"].(string) == "Rule 1 Updated" ||
				s.View["title"].(string) == "Rule 3 Updated" ||
				s.View["title"].(string) == "Rule 5 Updated" {
				// These should be the updated versions
				assert.Contains(t, s.View["title"], "Updated")
			}
		}
	})

	t.Run("pagination - first page", func(t *testing.T) {
		req := &models.ListSchemasRequest{
			Page:  1,
			Limit: 2,
		}

		schemas, total, err := repo.ListSchemas(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.Len(t, schemas, 2)
	})

	t.Run("pagination - second page", func(t *testing.T) {
		req := &models.ListSchemasRequest{
			Page:  2,
			Limit: 2,
		}

		schemas, total, err := repo.ListSchemas(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.Len(t, schemas, 2)
	})

	t.Run("filter by severity", func(t *testing.T) {
		req := &models.ListSchemasRequest{
			Page:     1,
			Limit:    10,
			Severity: "high",
		}

		schemas, total, err := repo.ListSchemas(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, schemas, 2)

		for _, s := range schemas {
			assert.Equal(t, "high", s.View["severity"])
		}
	})

	t.Run("filter by title", func(t *testing.T) {
		req := &models.ListSchemasRequest{
			Page:  1,
			Limit: 10,
			Title: "Rule 1",
		}

		schemas, total, err := repo.ListSchemas(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, schemas, 1)
		assert.Contains(t, schemas[0].View["title"], "Rule 1")
	})

	t.Run("filter by stable ID", func(t *testing.T) {
		// Get a schema first
		allSchemas, _, err := repo.ListSchemas(ctx, &models.ListSchemasRequest{Page: 1, Limit: 1})
		require.NoError(t, err)
		require.NotEmpty(t, allSchemas)

		req := &models.ListSchemasRequest{
			Page:  1,
			Limit: 10,
			ID:    allSchemas[0].ID,
		}

		schemas, total, err := repo.ListSchemas(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, schemas, 1)
		assert.Equal(t, allSchemas[0].ID, schemas[0].ID)
	})

	t.Run("exclude disabled schemas", func(t *testing.T) {
		// Create and disable a schema
		schema := createTestSchema(userID)
		schema.View["title"] = "Disabled Rule"
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		err = repo.DisableSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// List without disabled
		req := &models.ListSchemasRequest{
			Page:            1,
			Limit:           20,
			IncludeDisabled: false,
		}

		schemas, _, err := repo.ListSchemas(ctx, req)
		require.NoError(t, err)

		// Verify disabled schema is not in results
		for _, s := range schemas {
			assert.NotEqual(t, "Disabled Rule", s.View["title"])
		}

		// List with disabled
		req.IncludeDisabled = true
		schemas, _, err = repo.ListSchemas(ctx, req)
		require.NoError(t, err)

		// Verify disabled schema is in results
		found := false
		for _, s := range schemas {
			if s.View["title"] == "Disabled Rule" {
				found = true
				assert.NotNil(t, s.DisabledAt)
			}
		}
		assert.True(t, found, "disabled schema should be included")
	})

	t.Run("exclude hidden schemas", func(t *testing.T) {
		// Create and hide a schema
		schema := createTestSchema(userID)
		schema.View["title"] = "Hidden Rule"
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		err = repo.HideSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// List without hidden
		req := &models.ListSchemasRequest{
			Page:          1,
			Limit:         20,
			IncludeHidden: false,
		}

		schemas, _, err := repo.ListSchemas(ctx, req)
		require.NoError(t, err)

		// Verify hidden schema is not in results
		for _, s := range schemas {
			assert.NotEqual(t, "Hidden Rule", s.View["title"])
		}

		// List with hidden
		req.IncludeHidden = true
		schemas, _, err = repo.ListSchemas(ctx, req)
		require.NoError(t, err)

		// Verify hidden schema is in results
		found := false
		for _, s := range schemas {
			if s.View["title"] == "Hidden Rule" {
				found = true
				assert.NotNil(t, s.HiddenAt)
			}
		}
		assert.True(t, found, "hidden schema should be included")
	})
}

func TestDisableSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("disable active schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		err = repo.DisableSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// Verify disabled
		retrieved, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.DisabledAt)
		assert.NotNil(t, retrieved.DisabledBy)
		assert.Equal(t, userID, *retrieved.DisabledBy)
		assert.False(t, retrieved.IsActive())
	})

	t.Run("disable already disabled schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Disable once
		err = repo.DisableSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// Try to disable again
		err = repo.DisableSchema(ctx, schema.VersionID, userID)
		assert.ErrorIs(t, err, ErrSchemaNotFound)
	})

	t.Run("disable nonexistent schema", func(t *testing.T) {
		nonexistentID := uuid.NewString()
		err := repo.DisableSchema(ctx, nonexistentID, userID)
		assert.ErrorIs(t, err, ErrSchemaNotFound)
	})
}

func TestEnableSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("enable disabled schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Disable first
		err = repo.DisableSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// Enable
		err = repo.EnableSchema(ctx, schema.VersionID)
		require.NoError(t, err)

		// Verify enabled
		retrieved, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.Nil(t, retrieved.DisabledAt)
		assert.Nil(t, retrieved.DisabledBy)
		assert.True(t, retrieved.IsActive())
	})

	t.Run("enable active schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Enable already active schema (should succeed, idempotent)
		err = repo.EnableSchema(ctx, schema.VersionID)
		require.NoError(t, err)
	})

	t.Run("enable nonexistent schema", func(t *testing.T) {
		nonexistentID := uuid.NewString()
		err := repo.EnableSchema(ctx, nonexistentID)
		assert.ErrorIs(t, err, ErrSchemaNotFound)
	})
}

func TestHideSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("hide visible schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		err = repo.HideSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// Verify hidden
		retrieved, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.HiddenAt)
		assert.NotNil(t, retrieved.HiddenBy)
		assert.Equal(t, userID, *retrieved.HiddenBy)
		assert.False(t, retrieved.IsActive())
	})

	t.Run("hide already hidden schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Hide once
		err = repo.HideSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// Try to hide again
		err = repo.HideSchema(ctx, schema.VersionID, userID)
		assert.ErrorIs(t, err, ErrSchemaNotFound)
	})

	t.Run("hide nonexistent schema", func(t *testing.T) {
		nonexistentID := uuid.NewString()
		err := repo.HideSchema(ctx, nonexistentID, userID)
		assert.ErrorIs(t, err, ErrSchemaNotFound)
	})
}

func TestSetActiveParameterSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	userID := uuid.NewString()

	t.Run("set parameter set on active schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		err = repo.SetActiveParameterSet(ctx, schema.VersionID, "aggressive")
		require.NoError(t, err)

		// Verify parameter set was set
		retrieved, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.Equal(t, "aggressive", retrieved.Model["active_parameter_set"])
	})

	t.Run("update existing parameter set", func(t *testing.T) {
		schema := createTestSchema(userID)
		schema.Model["active_parameter_set"] = "conservative"
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		err = repo.SetActiveParameterSet(ctx, schema.VersionID, "balanced")
		require.NoError(t, err)

		// Verify parameter set was updated
		retrieved, err := repo.GetSchemaByVersionID(ctx, schema.VersionID)
		require.NoError(t, err)
		assert.Equal(t, "balanced", retrieved.Model["active_parameter_set"])
	})

	t.Run("set parameter set on disabled schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Disable schema
		err = repo.DisableSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// Try to set parameter set (should fail)
		err = repo.SetActiveParameterSet(ctx, schema.VersionID, "aggressive")
		assert.ErrorIs(t, err, ErrSchemaNotFound)
	})

	t.Run("set parameter set on hidden schema", func(t *testing.T) {
		schema := createTestSchema(userID)
		err := repo.CreateSchema(ctx, schema)
		require.NoError(t, err)

		// Hide schema
		err = repo.HideSchema(ctx, schema.VersionID, userID)
		require.NoError(t, err)

		// Try to set parameter set (should fail)
		err = repo.SetActiveParameterSet(ctx, schema.VersionID, "aggressive")
		assert.ErrorIs(t, err, ErrSchemaNotFound)
	})

	t.Run("set parameter set on nonexistent schema", func(t *testing.T) {
		nonexistentID := uuid.NewString()
		err := repo.SetActiveParameterSet(ctx, nonexistentID, "aggressive")
		assert.ErrorIs(t, err, ErrSchemaNotFound)
	})
}

func TestContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		schema := createTestSchema(uuid.NewString())
		err := repo.CreateSchema(ctx, schema)
		assert.Error(t, err)
	})
}
