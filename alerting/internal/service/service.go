package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/models"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/repository"
)

// Service handles business logic for cases
type Service struct {
	repo repository.Repository
}

// NewService creates a new service instance
func NewService(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

// CreateCase creates a new case
func (s *Service) CreateCase(ctx context.Context, req *models.CreateCaseRequest, userID string) (*models.Case, error) {
	// Validate severity
	if !isValidSeverity(req.Severity) {
		return nil, fmt.Errorf("invalid severity: %s", req.Severity)
	}

	caseUUID, _ := uuid.NewV7()
	c := &models.Case{
		ID:          caseUUID.String(),
		Title:       req.Title,
		Description: req.Description,
		Severity:    req.Severity,
		Status:      "open", // Default status
		Assignee:    req.Assignee,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateCase(ctx, c); err != nil {
		return nil, err
	}

	// Retrieve created case
	return s.repo.GetCaseByID(ctx, c.ID)
}

// GetCase retrieves a case by ID
func (s *Service) GetCase(ctx context.Context, id string) (*models.Case, error) {
	return s.repo.GetCaseByID(ctx, id)
}

// ListCases retrieves a paginated list of cases
func (s *Service) ListCases(ctx context.Context, req *models.ListCasesRequest) (*models.ListCasesResponse, error) {
	// Validate and set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 50
	}

	cases, total, err := s.repo.ListCases(ctx, req)
	if err != nil {
		return nil, err
	}

	totalPages := (total + req.Limit - 1) / req.Limit

	return &models.ListCasesResponse{
		Cases: cases,
		Pagination: models.Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateCase updates case fields
func (s *Service) UpdateCase(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) (*models.Case, error) {
	// Validate severity if provided
	if req.Severity != nil && !isValidSeverity(*req.Severity) {
		return nil, fmt.Errorf("invalid severity: %s", *req.Severity)
	}

	// Validate status if provided
	if req.Status != nil && !isValidStatus(*req.Status) {
		return nil, fmt.Errorf("invalid status: %s", *req.Status)
	}

	if err := s.repo.UpdateCase(ctx, id, req, userID); err != nil {
		return nil, err
	}

	return s.repo.GetCaseByID(ctx, id)
}

// CloseCase closes a case
func (s *Service) CloseCase(ctx context.Context, id string, userID string) error {
	return s.repo.CloseCase(ctx, id, userID)
}

// ReopenCase reopens a closed case
func (s *Service) ReopenCase(ctx context.Context, id string) error {
	return s.repo.ReopenCase(ctx, id)
}

// AddAlertsToCase adds alerts to a case
func (s *Service) AddAlertsToCase(ctx context.Context, caseID string, req *models.AddAlertsToCaseRequest, userID string) error {
	if len(req.AlertIDs) == 0 {
		return fmt.Errorf("no alert IDs provided")
	}

	return s.repo.AddAlertsToCase(ctx, caseID, req.AlertIDs, userID)
}

// GetCaseAlerts retrieves all alerts for a case
func (s *Service) GetCaseAlerts(ctx context.Context, caseID string) ([]*models.CaseAlert, error) {
	return s.repo.GetCaseAlerts(ctx, caseID)
}

// Helper functions for validation
func isValidSeverity(severity string) bool {
	validSeverities := map[string]bool{
		"info":     true,
		"low":      true,
		"medium":   true,
		"high":     true,
		"critical": true,
	}
	return validSeverities[severity]
}

func isValidStatus(status string) bool {
	validStatuses := map[string]bool{
		"open":        true,
		"in_progress": true,
		"resolved":    true,
		"closed":      true,
	}
	return validStatuses[status]
}
