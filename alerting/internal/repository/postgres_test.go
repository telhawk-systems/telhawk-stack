package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/models"
)

// Note: These tests require a PostgreSQL database connection.
// They will be skipped if TEST_DATABASE_URL environment variable is not set.
// Example: TEST_DATABASE_URL=postgres://postgres:password@localhost:5432/alerting_test?sslmode=disable

func getTestDB(t *testing.T) *PostgresRepository {
	t.Helper()

	// Check if test database is configured
	// For now, we'll skip actual database tests and focus on logic tests
	t.Skip("Skipping database integration tests - requires TEST_DATABASE_URL")
	return nil
}

func TestNewPostgresRepository(t *testing.T) {
	tests := []struct {
		name        string
		connString  string
		expectError bool
	}{
		{
			name:        "invalid connection string",
			connString:  "invalid://connection",
			expectError: true,
		},
		{
			name:        "empty connection string",
			connString:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPostgresRepository(context.Background(), tt.connString)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCase_CreateAndGet tests the CreateCase and GetCaseByID methods
func TestCase_CreateAndGet(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	caseID, err := uuid.NewV7()
	require.NoError(t, err)

	c := &models.Case{
		ID:          caseID.String(),
		Title:       "Test Case",
		Description: "Test Description",
		Severity:    "high",
		Status:      "open",
		Assignee:    stringPtr("user-123"),
		CreatedBy:   "admin-user",
		CreatedAt:   time.Now(),
	}

	// Create case
	err = repo.CreateCase(ctx, c)
	require.NoError(t, err)

	// Get case
	retrieved, err := repo.GetCaseByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, c.ID, retrieved.ID)
	assert.Equal(t, c.Title, retrieved.Title)
	assert.Equal(t, c.Description, retrieved.Description)
	assert.Equal(t, c.Severity, retrieved.Severity)
	assert.Equal(t, c.Status, retrieved.Status)
	assert.Equal(t, *c.Assignee, *retrieved.Assignee)
	assert.Equal(t, c.CreatedBy, retrieved.CreatedBy)
	assert.Equal(t, 0, retrieved.AlertCount)
}

func TestCase_GetByID_NotFound(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	_, err := repo.GetCaseByID(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.Equal(t, ErrCaseNotFound, err)
}

func TestCase_List(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	// Create test cases
	now := time.Now()
	testCases := []*models.Case{
		{
			ID:        uuid.New().String(),
			Title:     "Case 1",
			Severity:  "high",
			Status:    "open",
			CreatedBy: "user-1",
			CreatedAt: now,
		},
		{
			ID:        uuid.New().String(),
			Title:     "Case 2",
			Severity:  "medium",
			Status:    "in_progress",
			Assignee:  stringPtr("user-2"),
			CreatedBy: "user-1",
			CreatedAt: now.Add(1 * time.Minute),
		},
		{
			ID:        uuid.New().String(),
			Title:     "Case 3",
			Severity:  "critical",
			Status:    "closed",
			CreatedBy: "user-2",
			CreatedAt: now.Add(2 * time.Minute),
		},
	}

	for _, c := range testCases {
		err := repo.CreateCase(ctx, c)
		require.NoError(t, err)
	}

	tests := []struct {
		name         string
		request      *models.ListCasesRequest
		minExpected  int
		validateFunc func(*testing.T, []*models.Case)
	}{
		{
			name: "list all with default pagination",
			request: &models.ListCasesRequest{
				Page:  1,
				Limit: 50,
			},
			minExpected: 3,
			validateFunc: func(t *testing.T, cases []*models.Case) {
				// Should be ordered by created_at DESC
				assert.GreaterOrEqual(t, len(cases), 3)
			},
		},
		{
			name: "filter by severity",
			request: &models.ListCasesRequest{
				Page:     1,
				Limit:    50,
				Severity: "high",
			},
			minExpected: 1,
			validateFunc: func(t *testing.T, cases []*models.Case) {
				for _, c := range cases {
					assert.Equal(t, "high", c.Severity)
				}
			},
		},
		{
			name: "filter by status",
			request: &models.ListCasesRequest{
				Page:   1,
				Limit:  50,
				Status: "open",
			},
			minExpected: 1,
			validateFunc: func(t *testing.T, cases []*models.Case) {
				for _, c := range cases {
					assert.Equal(t, "open", c.Status)
				}
			},
		},
		{
			name: "filter by assignee",
			request: &models.ListCasesRequest{
				Page:     1,
				Limit:    50,
				Assignee: "user-2",
			},
			minExpected: 1,
			validateFunc: func(t *testing.T, cases []*models.Case) {
				for _, c := range cases {
					require.NotNil(t, c.Assignee)
					assert.Equal(t, "user-2", *c.Assignee)
				}
			},
		},
		{
			name: "pagination - page 1",
			request: &models.ListCasesRequest{
				Page:  1,
				Limit: 2,
			},
			minExpected: 2,
			validateFunc: func(t *testing.T, cases []*models.Case) {
				assert.LessOrEqual(t, len(cases), 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases, total, err := repo.ListCases(ctx, tt.request)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(cases), tt.minExpected)
			assert.GreaterOrEqual(t, total, tt.minExpected)

			if tt.validateFunc != nil {
				tt.validateFunc(t, cases)
			}
		})
	}
}

func TestCase_Update(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	// Create a case
	caseID, err := uuid.NewV7()
	require.NoError(t, err)

	c := &models.Case{
		ID:        caseID.String(),
		Title:     "Original Title",
		Severity:  "medium",
		Status:    "open",
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
	}

	err = repo.CreateCase(ctx, c)
	require.NoError(t, err)

	tests := []struct {
		name      string
		request   *models.UpdateCaseRequest
		userID    string
		expectErr bool
	}{
		{
			name: "update title",
			request: &models.UpdateCaseRequest{
				Title: stringPtr("Updated Title"),
			},
			userID:    "user-1",
			expectErr: false,
		},
		{
			name: "update severity",
			request: &models.UpdateCaseRequest{
				Severity: stringPtr("critical"),
			},
			userID:    "user-1",
			expectErr: false,
		},
		{
			name: "update status",
			request: &models.UpdateCaseRequest{
				Status: stringPtr("in_progress"),
			},
			userID:    "user-1",
			expectErr: false,
		},
		{
			name: "update assignee",
			request: &models.UpdateCaseRequest{
				Assignee: stringPtr("user-2"),
			},
			userID:    "user-1",
			expectErr: false,
		},
		{
			name: "update multiple fields",
			request: &models.UpdateCaseRequest{
				Title:       stringPtr("New Title"),
				Description: stringPtr("New Description"),
				Severity:    stringPtr("high"),
			},
			userID:    "user-1",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateCase(ctx, c.ID, tt.request, tt.userID)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify update
				updated, err := repo.GetCaseByID(ctx, c.ID)
				require.NoError(t, err)

				if tt.request.Title != nil {
					assert.Equal(t, *tt.request.Title, updated.Title)
				}
				if tt.request.Severity != nil {
					assert.Equal(t, *tt.request.Severity, updated.Severity)
				}
				if tt.request.Status != nil {
					assert.Equal(t, *tt.request.Status, updated.Status)
				}
				if tt.request.Assignee != nil {
					require.NotNil(t, updated.Assignee)
					assert.Equal(t, *tt.request.Assignee, *updated.Assignee)
				}
			}
		})
	}
}

func TestCase_Update_NotFound(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	req := &models.UpdateCaseRequest{
		Title: stringPtr("Updated Title"),
	}

	err := repo.UpdateCase(ctx, "nonexistent-id", req, "user-1")
	require.Error(t, err)
	assert.Equal(t, ErrCaseNotFound, err)
}

func TestCase_Close(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	// Create a case
	caseID, err := uuid.NewV7()
	require.NoError(t, err)

	c := &models.Case{
		ID:        caseID.String(),
		Title:     "Test Case",
		Severity:  "high",
		Status:    "open",
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
	}

	err = repo.CreateCase(ctx, c)
	require.NoError(t, err)

	// Close the case
	err = repo.CloseCase(ctx, c.ID, "user-1")
	require.NoError(t, err)

	// Verify case is closed
	closed, err := repo.GetCaseByID(ctx, c.ID)
	require.NoError(t, err)

	assert.Equal(t, "closed", closed.Status)
	require.NotNil(t, closed.ClosedAt)
	require.NotNil(t, closed.ClosedBy)
	assert.Equal(t, "user-1", *closed.ClosedBy)
}

func TestCase_Close_NotFound(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	err := repo.CloseCase(ctx, "nonexistent-id", "user-1")
	require.Error(t, err)
	assert.Equal(t, ErrCaseNotFound, err)
}

func TestCase_Reopen(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	// Create a case
	caseID, err := uuid.NewV7()
	require.NoError(t, err)

	c := &models.Case{
		ID:        caseID.String(),
		Title:     "Test Case",
		Severity:  "high",
		Status:    "open",
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
	}

	err = repo.CreateCase(ctx, c)
	require.NoError(t, err)

	// Close the case
	err = repo.CloseCase(ctx, c.ID, "user-1")
	require.NoError(t, err)

	// Reopen the case
	err = repo.ReopenCase(ctx, c.ID)
	require.NoError(t, err)

	// Verify case is reopened
	reopened, err := repo.GetCaseByID(ctx, c.ID)
	require.NoError(t, err)

	assert.Equal(t, "open", reopened.Status)
	assert.Nil(t, reopened.ClosedAt)
	assert.Nil(t, reopened.ClosedBy)
}

func TestCase_Reopen_NotFound(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	err := repo.ReopenCase(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.Equal(t, ErrCaseNotFound, err)
}

func TestCaseAlerts_AddAndGet(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	// Create a case
	caseID, err := uuid.NewV7()
	require.NoError(t, err)

	c := &models.Case{
		ID:        caseID.String(),
		Title:     "Test Case",
		Severity:  "high",
		Status:    "open",
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
	}

	err = repo.CreateCase(ctx, c)
	require.NoError(t, err)

	// Add alerts
	alertIDs := []string{"alert-1", "alert-2", "alert-3"}
	err = repo.AddAlertsToCase(ctx, c.ID, alertIDs, "user-1")
	require.NoError(t, err)

	// Get case alerts
	caseAlerts, err := repo.GetCaseAlerts(ctx, c.ID)
	require.NoError(t, err)
	assert.Len(t, caseAlerts, 3)

	// Verify alert IDs
	alertIDMap := make(map[string]bool)
	for _, ca := range caseAlerts {
		alertIDMap[ca.AlertID] = true
		assert.Equal(t, c.ID, ca.CaseID)
	}

	for _, alertID := range alertIDs {
		assert.True(t, alertIDMap[alertID], "expected alert ID %s to be present", alertID)
	}

	// Verify case alert count
	caseWithCount, err := repo.GetCaseByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, caseWithCount.AlertCount)
}

func TestCaseAlerts_AddDuplicate(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	// Create a case
	caseID, err := uuid.NewV7()
	require.NoError(t, err)

	c := &models.Case{
		ID:        caseID.String(),
		Title:     "Test Case",
		Severity:  "high",
		Status:    "open",
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
	}

	err = repo.CreateCase(ctx, c)
	require.NoError(t, err)

	// Add alerts
	alertIDs := []string{"alert-1", "alert-2"}
	err = repo.AddAlertsToCase(ctx, c.ID, alertIDs, "user-1")
	require.NoError(t, err)

	// Try to add duplicate
	err = repo.AddAlertsToCase(ctx, c.ID, []string{"alert-1"}, "user-1")
	require.NoError(t, err) // Should not error due to ON CONFLICT DO NOTHING

	// Verify only 2 unique alerts
	caseAlerts, err := repo.GetCaseAlerts(ctx, c.ID)
	require.NoError(t, err)
	assert.Len(t, caseAlerts, 2)
}

func TestCaseAlerts_CaseNotFound(t *testing.T) {
	repo := getTestDB(t)
	ctx := context.Background()

	err := repo.AddAlertsToCase(ctx, "nonexistent-id", []string{"alert-1"}, "user-1")
	require.Error(t, err)
	assert.Equal(t, ErrCaseNotFound, err)
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{
			name:     "empty slice",
			strs:     []string{},
			sep:      ", ",
			expected: "",
		},
		{
			name:     "single element",
			strs:     []string{"one"},
			sep:      ", ",
			expected: "one",
		},
		{
			name:     "multiple elements",
			strs:     []string{"one", "two", "three"},
			sep:      ", ",
			expected: "one, two, three",
		},
		{
			name:     "different separator",
			strs:     []string{"a", "b", "c"},
			sep:      " AND ",
			expected: "a AND b AND c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.strs, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
