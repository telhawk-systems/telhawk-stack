package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/models"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/service"
)

// mockRepository is a mock implementation of repository.Repository for testing handlers
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

// mockStorageClient is a mock implementation of StorageClient
type mockStorageClient struct {
	queryFunc func(method, path string, body []byte) ([]byte, error)
}

func (m *mockStorageClient) Query(method, path string, body []byte) ([]byte, error) {
	if m.queryFunc != nil {
		return m.queryFunc(method, path, body)
	}
	return nil, errors.New("not implemented")
}

func TestHealthCheck(t *testing.T) {
	mockRepo := &mockRepository{}
	mockSvc := service.NewService(mockRepo)
	mockStorage := &mockStorageClient{}
	h := NewHandler(mockSvc, mockStorage)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestCreateCase(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		setupMock      func(*mockRepository)
		expectedStatus int
	}{
		{
			name:   "successful case creation",
			method: http.MethodPost,
			requestBody: models.CreateCaseRequest{
				Title:       "Test Case",
				Description: "Test Description",
				Severity:    "high",
			},
			setupMock: func(m *mockRepository) {
				m.createCaseFunc = func(ctx context.Context, c *models.Case) error {
					return nil
				}
				m.getCaseByIDFunc = func(ctx context.Context, id string) (*models.Case, error) {
					return &models.Case{
						ID:          id,
						Title:       "Test Case",
						Description: "Test Description",
						Severity:    "high",
						Status:      "open",
						CreatedBy:   "00000000-0000-0000-0000-000000000001",
						CreatedAt:   time.Now(),
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "method not allowed",
			method:         http.MethodGet,
			requestBody:    nil,
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "invalid request body",
			method:         http.MethodPost,
			requestBody:    "invalid json",
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			method: http.MethodPost,
			requestBody: models.CreateCaseRequest{
				Title:    "Test Case",
				Severity: "high",
			},
			setupMock: func(m *mockRepository) {
				m.createCaseFunc = func(ctx context.Context, c *models.Case) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)
			mockSvc := service.NewService(mockRepo)
			mockStorage := &mockStorageClient{}
			h := NewHandler(mockSvc, mockStorage)

			var body []byte
			if tt.requestBody != nil {
				if str, ok := tt.requestBody.(string); ok {
					body = []byte(str)
				} else {
					body, _ = json.Marshal(tt.requestBody)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/v1/cases", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.CreateCase(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response models.Case
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
			}
		})
	}
}

func TestGetCase(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		setupMock      func(*mockRepository)
		expectedStatus int
	}{
		{
			name:   "successful case retrieval",
			method: http.MethodGet,
			path:   "/api/v1/cases/case-123",
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
			expectedStatus: http.StatusOK,
		},
		{
			name:           "method not allowed",
			method:         http.MethodPost,
			path:           "/api/v1/cases/case-123",
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "missing case ID",
			method:         http.MethodGet,
			path:           "/api/v1/cases/",
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "case not found",
			method: http.MethodGet,
			path:   "/api/v1/cases/nonexistent",
			setupMock: func(m *mockRepository) {
				m.getCaseByIDFunc = func(ctx context.Context, id string) (*models.Case, error) {
					return nil, repository.ErrCaseNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)
			mockSvc := service.NewService(mockRepo)
			mockStorage := &mockStorageClient{}
			h := NewHandler(mockSvc, mockStorage)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			h.GetCase(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.Case
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ID)
			}
		})
	}
}

func TestListCases(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		queryParams    string
		setupMock      func(*mockRepository)
		expectedStatus int
	}{
		{
			name:        "successful list with defaults",
			method:      http.MethodGet,
			queryParams: "",
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					return []*models.Case{
						{ID: "case-1", Title: "Case 1"},
						{ID: "case-2", Title: "Case 2"},
					}, 2, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "list with pagination",
			method:      http.MethodGet,
			queryParams: "?page=2&limit=10",
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					return []*models.Case{}, 0, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "method not allowed",
			method:         http.MethodPost,
			queryParams:    "",
			setupMock:      func(m *mockRepository) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:        "service error",
			method:      http.MethodGet,
			queryParams: "",
			setupMock: func(m *mockRepository) {
				m.listCasesFunc = func(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
					return nil, 0, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tt.setupMock(mockRepo)
			mockSvc := service.NewService(mockRepo)
			mockStorage := &mockStorageClient{}
			h := NewHandler(mockSvc, mockStorage)

			req := httptest.NewRequest(tt.method, "/api/v1/cases"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			h.ListCases(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.ListCasesResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
			}
		})
	}
}

func TestListAlerts(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		queryParams    string
		setupMock      func(*mockStorageClient)
		expectedStatus int
	}{
		{
			name:        "successful list",
			method:      http.MethodGet,
			queryParams: "",
			setupMock: func(m *mockStorageClient) {
				m.queryFunc = func(method, path string, body []byte) ([]byte, error) {
					response := map[string]interface{}{
						"hits": map[string]interface{}{
							"total": map[string]interface{}{"value": 2},
							"hits": []map[string]interface{}{
								{
									"_id":    "alert-1",
									"_index": "telhawk-alerts-2024.01.01",
									"_source": map[string]interface{}{
										"severity": "high",
										"title":    "Alert 1",
									},
								},
							},
						},
					}
					return json.Marshal(response)
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "method not allowed",
			method:         http.MethodPost,
			queryParams:    "",
			setupMock:      func(m *mockStorageClient) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:        "storage error",
			method:      http.MethodGet,
			queryParams: "",
			setupMock: func(m *mockStorageClient) {
				m.queryFunc = func(method, path string, body []byte) ([]byte, error) {
					return nil, errors.New("storage error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			mockSvc := service.NewService(mockRepo)
			mockStorage := &mockStorageClient{}
			tt.setupMock(mockStorage)
			h := NewHandler(mockSvc, mockStorage)

			req := httptest.NewRequest(tt.method, "/api/v1/alerts"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			h.ListAlerts(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "alerts")
				assert.Contains(t, response, "total")
			}
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
