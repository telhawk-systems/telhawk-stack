package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/models"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/repository"
)

// mockRepository is a mock implementation of repository.Repository
type mockRepository struct {
	createCaseFunc      func(ctx context.Context, c *models.Case) error
	getCaseByIDFunc     func(ctx context.Context, id string) (*models.Case, error)
	listCasesFunc       func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error)
	updateCaseFunc      func(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) error
	closeCaseFunc       func(ctx context.Context, id string, userID string) error
	reopenCaseFunc      func(ctx context.Context, id string) error
	addAlertsToCaseFunc func(ctx context.Context, caseID string, alertIDs []string, userID string) error
	getCaseAlertsFunc   func(ctx context.Context, caseID string) ([]*models.CaseAlert, error)
}

func (m *mockRepository) CreateCase(ctx context.Context, c *models.Case) error {
	if m.createCaseFunc != nil {
		return m.createCaseFunc(ctx, c)
	}
	return nil
}

func (m *mockRepository) GetCaseByID(ctx context.Context, id string) (*models.Case, error) {
	if m.getCaseByIDFunc != nil {
		return m.getCaseByIDFunc(ctx, id)
	}
	return nil, repository.ErrCaseNotFound
}

func (m *mockRepository) ListCases(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
	if m.listCasesFunc != nil {
		return m.listCasesFunc(ctx, req)
	}
	return nil, 0, nil
}

func (m *mockRepository) UpdateCase(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) error {
	if m.updateCaseFunc != nil {
		return m.updateCaseFunc(ctx, id, req, userID)
	}
	return nil
}

func (m *mockRepository) CloseCase(ctx context.Context, id string, userID string) error {
	if m.closeCaseFunc != nil {
		return m.closeCaseFunc(ctx, id, userID)
	}
	return nil
}

func (m *mockRepository) ReopenCase(ctx context.Context, id string) error {
	if m.reopenCaseFunc != nil {
		return m.reopenCaseFunc(ctx, id)
	}
	return nil
}

func (m *mockRepository) AddAlertsToCase(ctx context.Context, caseID string, alertIDs []string, userID string) error {
	if m.addAlertsToCaseFunc != nil {
		return m.addAlertsToCaseFunc(ctx, caseID, alertIDs, userID)
	}
	return nil
}

func (m *mockRepository) GetCaseAlerts(ctx context.Context, caseID string) ([]*models.CaseAlert, error) {
	if m.getCaseAlertsFunc != nil {
		return m.getCaseAlertsFunc(ctx, caseID)
	}
	return nil, nil
}

func (m *mockRepository) Close() error {
	return nil
}

func TestCreateCase(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.CreateCaseRequest
		userID      string
		setupMock   func(*mockRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful case creation",
			request: &models.CreateCaseRequest{
				Title:       "Test Case",
				Description: "Test Description",
				Severity:    "high",
			},
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.createCaseFunc = func(ctx context.Context, c *models.Case) error {
					assert.Equal(t, "Test Case", c.Title)
					assert.Equal(t, "Test Description", c.Description)
					assert.Equal(t, "high", c.Severity)
					assert.Equal(t, "open", c.Status)
					assert.Equal(t, "user-123", c.CreatedBy)
					return nil
				}
				m.getCaseByIDFunc = func(ctx context.Context, id string) (*models.Case, error) {
					return &models.Case{
						ID:          id,
						Title:       "Test Case",
						Description: "Test Description",
						Severity:    "high",
						Status:      "open",
						CreatedBy:   "user-123",
						CreatedAt:   time.Now(),
					}, nil
				}
			},
			expectError: false,
		},
		{
			name: "invalid severity",
			request: &models.CreateCaseRequest{
				Title:    "Test Case",
				Severity: "invalid",
			},
			userID:      "user-123",
			setupMock:   func(m *mockRepository) {},
			expectError: true,
			errorMsg:    "invalid severity: invalid",
		},
		{
			name: "valid severity levels",
			request: &models.CreateCaseRequest{
				Title:    "Test Case",
				Severity: "critical",
			},
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.createCaseFunc = func(ctx context.Context, c *models.Case) error {
					return nil
				}
				m.getCaseByIDFunc = func(ctx context.Context, id string) (*models.Case, error) {
					return &models.Case{ID: id, Severity: "critical"}, nil
				}
			},
			expectError: false,
		},
		{
			name: "repository error",
			request: &models.CreateCaseRequest{
				Title:    "Test Case",
				Severity: "high",
			},
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.createCaseFunc = func(ctx context.Context, c *models.Case) error {
					return errors.New("database error")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo)
			c, err := svc.CreateCase(context.Background(), tt.request, tt.userID)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, c)
				assert.NotEmpty(t, c.ID)
			}
		})
	}
}

func TestGetCase(t *testing.T) {
	tests := []struct {
		name        string
		caseID      string
		setupMock   func(*mockRepository)
		expectError bool
	}{
		{
			name:   "successful case retrieval",
			caseID: "case-123",
			setupMock: func(m *mockRepository) {
				m.getCaseByIDFunc = func(ctx context.Context, id string) (*models.Case, error) {
					return &models.Case{
						ID:       id,
						Title:    "Test Case",
						Severity: "high",
						Status:   "open",
					}, nil
				}
			},
			expectError: false,
		},
		{
			name:   "case not found",
			caseID: "nonexistent",
			setupMock: func(m *mockRepository) {
				m.getCaseByIDFunc = func(ctx context.Context, id string) (*models.Case, error) {
					return nil, repository.ErrCaseNotFound
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo)
			c, err := svc.GetCase(context.Background(), tt.caseID)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, c)
				assert.Equal(t, tt.caseID, c.ID)
			}
		})
	}
}

func TestListCases(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.ListCasesRequest
		setupMock     func(*mockRepository)
		expectedCount int
		expectedPages int
		expectedTotal int
		expectError   bool
	}{
		{
			name: "successful list with defaults",
			request: &models.ListCasesRequest{
				Page:  1,
				Limit: 50,
			},
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					return []*models.Case{
						{ID: "case-1", Title: "Case 1"},
						{ID: "case-2", Title: "Case 2"},
					}, 2, nil
				}
			},
			expectedCount: 2,
			expectedPages: 1,
			expectedTotal: 2,
			expectError:   false,
		},
		{
			name: "pagination validation - invalid page",
			request: &models.ListCasesRequest{
				Page:  0,
				Limit: 50,
			},
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					// Should be normalized to page 1
					assert.Equal(t, 1, req.Page)
					return []*models.Case{}, 0, nil
				}
			},
			expectedCount: 0,
			expectedPages: 0,
			expectedTotal: 0,
			expectError:   false,
		},
		{
			name: "pagination validation - invalid limit",
			request: &models.ListCasesRequest{
				Page:  1,
				Limit: 0,
			},
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					// Should be normalized to limit 50
					assert.Equal(t, 50, req.Limit)
					return []*models.Case{}, 0, nil
				}
			},
			expectedCount: 0,
			expectedPages: 0,
			expectedTotal: 0,
			expectError:   false,
		},
		{
			name: "pagination validation - limit too high",
			request: &models.ListCasesRequest{
				Page:  1,
				Limit: 200,
			},
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					// Should be normalized to limit 50
					assert.Equal(t, 50, req.Limit)
					return []*models.Case{}, 0, nil
				}
			},
			expectedCount: 0,
			expectedPages: 0,
			expectedTotal: 0,
			expectError:   false,
		},
		{
			name: "filtered by severity",
			request: &models.ListCasesRequest{
				Page:     1,
				Limit:    50,
				Severity: "high",
			},
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					assert.Equal(t, "high", req.Severity)
					return []*models.Case{
						{ID: "case-1", Severity: "high"},
					}, 1, nil
				}
			},
			expectedCount: 1,
			expectedPages: 1,
			expectedTotal: 1,
			expectError:   false,
		},
		{
			name: "repository error",
			request: &models.ListCasesRequest{
				Page:  1,
				Limit: 50,
			},
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					return nil, 0, errors.New("database error")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo)
			resp, err := svc.ListCases(context.Background(), tt.request)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Len(t, resp.Cases, tt.expectedCount)
				assert.Equal(t, tt.expectedPages, resp.Pagination.TotalPages)
				assert.Equal(t, tt.expectedTotal, resp.Pagination.Total)
			}
		})
	}
}

func TestUpdateCase(t *testing.T) {
	tests := []struct {
		name        string
		caseID      string
		request     *models.UpdateCaseRequest
		userID      string
		setupMock   func(*mockRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name:   "successful update",
			caseID: "case-123",
			request: &models.UpdateCaseRequest{
				Title:       stringPtr("Updated Title"),
				Description: stringPtr("Updated Description"),
			},
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.updateCaseFunc = func(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) error {
					return nil
				}
				m.getCaseByIDFunc = func(ctx context.Context, id string) (*models.Case, error) {
					return &models.Case{
						ID:          id,
						Title:       "Updated Title",
						Description: "Updated Description",
					}, nil
				}
			},
			expectError: false,
		},
		{
			name:   "invalid severity",
			caseID: "case-123",
			request: &models.UpdateCaseRequest{
				Severity: stringPtr("invalid"),
			},
			userID:      "user-123",
			setupMock:   func(m *mockRepository) {},
			expectError: true,
			errorMsg:    "invalid severity: invalid",
		},
		{
			name:   "invalid status",
			caseID: "case-123",
			request: &models.UpdateCaseRequest{
				Status: stringPtr("invalid"),
			},
			userID:      "user-123",
			setupMock:   func(m *mockRepository) {},
			expectError: true,
			errorMsg:    "invalid status: invalid",
		},
		{
			name:   "case not found",
			caseID: "nonexistent",
			request: &models.UpdateCaseRequest{
				Title: stringPtr("Updated Title"),
			},
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.updateCaseFunc = func(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) error {
					return repository.ErrCaseNotFound
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo)
			c, err := svc.UpdateCase(context.Background(), tt.caseID, tt.request, tt.userID)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, c)
			}
		})
	}
}

func TestCloseCase(t *testing.T) {
	tests := []struct {
		name        string
		caseID      string
		userID      string
		setupMock   func(*mockRepository)
		expectError bool
	}{
		{
			name:   "successful close",
			caseID: "case-123",
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.closeCaseFunc = func(ctx context.Context, id string, userID string) error {
					assert.Equal(t, "case-123", id)
					assert.Equal(t, "user-123", userID)
					return nil
				}
			},
			expectError: false,
		},
		{
			name:   "case not found",
			caseID: "nonexistent",
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.closeCaseFunc = func(ctx context.Context, id string, userID string) error {
					return repository.ErrCaseNotFound
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo)
			err := svc.CloseCase(context.Background(), tt.caseID, tt.userID)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReopenCase(t *testing.T) {
	tests := []struct {
		name        string
		caseID      string
		setupMock   func(*mockRepository)
		expectError bool
	}{
		{
			name:   "successful reopen",
			caseID: "case-123",
			setupMock: func(m *mockRepository) {
				m.reopenCaseFunc = func(ctx context.Context, id string) error {
					assert.Equal(t, "case-123", id)
					return nil
				}
			},
			expectError: false,
		},
		{
			name:   "case not found",
			caseID: "nonexistent",
			setupMock: func(m *mockRepository) {
				m.reopenCaseFunc = func(ctx context.Context, id string) error {
					return repository.ErrCaseNotFound
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo)
			err := svc.ReopenCase(context.Background(), tt.caseID)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAddAlertsToCase(t *testing.T) {
	tests := []struct {
		name        string
		caseID      string
		request     *models.AddAlertsToCaseRequest
		userID      string
		setupMock   func(*mockRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name:   "successful add alerts",
			caseID: "case-123",
			request: &models.AddAlertsToCaseRequest{
				AlertIDs: []string{"alert-1", "alert-2"},
			},
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.addAlertsToCaseFunc = func(ctx context.Context, caseID string, alertIDs []string, userID string) error {
					assert.Equal(t, "case-123", caseID)
					assert.Len(t, alertIDs, 2)
					assert.Equal(t, "user-123", userID)
					return nil
				}
			},
			expectError: false,
		},
		{
			name:   "empty alert IDs",
			caseID: "case-123",
			request: &models.AddAlertsToCaseRequest{
				AlertIDs: []string{},
			},
			userID:      "user-123",
			setupMock:   func(m *mockRepository) {},
			expectError: true,
			errorMsg:    "no alert IDs provided",
		},
		{
			name:   "repository error",
			caseID: "case-123",
			request: &models.AddAlertsToCaseRequest{
				AlertIDs: []string{"alert-1"},
			},
			userID: "user-123",
			setupMock: func(m *mockRepository) {
				m.addAlertsToCaseFunc = func(ctx context.Context, caseID string, alertIDs []string, userID string) error {
					return errors.New("database error")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo)
			err := svc.AddAlertsToCase(context.Background(), tt.caseID, tt.request, tt.userID)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetCaseAlerts(t *testing.T) {
	tests := []struct {
		name          string
		caseID        string
		setupMock     func(*mockRepository)
		expectedCount int
		expectError   bool
	}{
		{
			name:   "successful retrieval",
			caseID: "case-123",
			setupMock: func(m *mockRepository) {
				m.getCaseAlertsFunc = func(ctx context.Context, caseID string) ([]*models.CaseAlert, error) {
					return []*models.CaseAlert{
						{CaseID: caseID, AlertID: "alert-1"},
						{CaseID: caseID, AlertID: "alert-2"},
					}, nil
				}
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:   "empty results",
			caseID: "case-123",
			setupMock: func(m *mockRepository) {
				m.getCaseAlertsFunc = func(ctx context.Context, caseID string) ([]*models.CaseAlert, error) {
					return []*models.CaseAlert{}, nil
				}
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:   "repository error",
			caseID: "case-123",
			setupMock: func(m *mockRepository) {
				m.getCaseAlertsFunc = func(ctx context.Context, caseID string) ([]*models.CaseAlert, error) {
					return nil, errors.New("database error")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo)
			alerts, err := svc.GetCaseAlerts(context.Background(), tt.caseID)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, alerts)
				assert.Len(t, alerts, tt.expectedCount)
			}
		})
	}
}

func TestValidationHelpers(t *testing.T) {
	t.Run("isValidSeverity", func(t *testing.T) {
		validSeverities := []string{"info", "low", "medium", "high", "critical"}
		for _, severity := range validSeverities {
			assert.True(t, isValidSeverity(severity), "expected %s to be valid", severity)
		}

		invalidSeverities := []string{"", "invalid", "CRITICAL", "High"}
		for _, severity := range invalidSeverities {
			assert.False(t, isValidSeverity(severity), "expected %s to be invalid", severity)
		}
	})

	t.Run("isValidStatus", func(t *testing.T) {
		validStatuses := []string{"open", "in_progress", "resolved", "closed"}
		for _, status := range validStatuses {
			assert.True(t, isValidStatus(status), "expected %s to be valid", status)
		}

		invalidStatuses := []string{"", "invalid", "OPEN", "Open"}
		for _, status := range invalidStatuses {
			assert.False(t, isValidStatus(status), "expected %s to be invalid", status)
		}
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
