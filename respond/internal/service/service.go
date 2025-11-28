// Package service provides business logic for the respond service.
package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/models"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/repository"
)

// Service provides business logic for the respond service
type Service struct {
	repo repository.Repository
}

// NewService creates a new Service instance
func NewService(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

// =============================================================================
// Detection Schema Methods
// =============================================================================

// CreateSchema creates a new detection schema
func (s *Service) CreateSchema(ctx context.Context, req *models.CreateSchemaRequest) (*models.DetectionSchema, error) {
	schema := &models.DetectionSchema{
		ID:         req.ID, // May be empty, repository will generate
		Model:      req.Model,
		View:       req.View,
		Controller: req.Controller,
	}

	// Generate IDs if not provided
	if schema.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema ID: %w", err)
		}
		schema.ID = id.String()
	}
	versionID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate version ID: %w", err)
	}
	schema.VersionID = versionID.String()

	if err := s.repo.CreateSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Fetch the created schema to get all fields
	return s.repo.GetSchemaByVersionID(ctx, schema.VersionID)
}

// GetSchema retrieves a detection schema by ID (stable ID returns latest, version_id returns specific)
func (s *Service) GetSchema(ctx context.Context, id string) (*models.DetectionSchema, error) {
	// Try to get by stable ID first (returns latest version)
	schema, err := s.repo.GetLatestSchemaByID(ctx, id)
	if err == nil {
		return schema, nil
	}

	// If not found by stable ID, try version_id
	if errors.Is(err, repository.ErrSchemaNotFound) {
		return s.repo.GetSchemaByVersionID(ctx, id)
	}

	return nil, err
}

// GetSchemaByVersionID retrieves a specific version of a detection schema
func (s *Service) GetSchemaByVersionID(ctx context.Context, versionID string) (*models.DetectionSchema, error) {
	return s.repo.GetSchemaByVersionID(ctx, versionID)
}

// ListSchemas retrieves a paginated list of detection schemas
func (s *Service) ListSchemas(ctx context.Context, req *models.ListSchemasRequest) (*models.ListSchemasResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	schemas, total, err := s.repo.ListSchemas(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	totalPages := (total + req.Limit - 1) / req.Limit

	return &models.ListSchemasResponse{
		Schemas: schemas,
		Pagination: models.Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateSchema creates a new version of an existing schema
func (s *Service) UpdateSchema(ctx context.Context, id string, req *models.UpdateSchemaRequest) (*models.DetectionSchema, error) {
	// Verify the schema exists
	_, err := s.repo.GetLatestSchemaByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing schema: %w", err)
	}

	// Create new version with same stable ID
	schema := &models.DetectionSchema{
		ID:         id, // Keep same stable ID
		Model:      req.Model,
		View:       req.View,
		Controller: req.Controller,
	}

	// Generate new version_id
	versionID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate version ID: %w", err)
	}
	schema.VersionID = versionID.String()

	if err := s.repo.CreateSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to create new version: %w", err)
	}

	return s.repo.GetSchemaByVersionID(ctx, schema.VersionID)
}

// DisableSchema disables a detection schema (stops evaluation)
func (s *Service) DisableSchema(ctx context.Context, id, userID string) (*models.DetectionSchema, error) {
	// Get the latest version to disable
	schema, err := s.repo.GetLatestSchemaByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.DisableSchema(ctx, schema.VersionID, userID); err != nil {
		return nil, fmt.Errorf("failed to disable schema: %w", err)
	}

	return s.repo.GetSchemaByVersionID(ctx, schema.VersionID)
}

// EnableSchema re-enables a disabled detection schema
func (s *Service) EnableSchema(ctx context.Context, id string) (*models.DetectionSchema, error) {
	// Get the latest version to enable
	schema, err := s.repo.GetLatestSchemaByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.EnableSchema(ctx, schema.VersionID); err != nil {
		return nil, fmt.Errorf("failed to enable schema: %w", err)
	}

	return s.repo.GetSchemaByVersionID(ctx, schema.VersionID)
}

// HideSchema hides (soft deletes) a detection schema
func (s *Service) HideSchema(ctx context.Context, id, userID string) error {
	// Get the latest version to hide
	schema, err := s.repo.GetLatestSchemaByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.HideSchema(ctx, schema.VersionID, userID)
}

// GetVersionHistory retrieves all versions of a detection schema
func (s *Service) GetVersionHistory(ctx context.Context, id string) (*models.VersionHistoryResponse, error) {
	versions, err := s.repo.GetSchemaVersionHistory(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get version history: %w", err)
	}

	// Get the title from the latest version
	title := ""
	if len(versions) > 0 {
		title = versions[0].Title
	}

	return &models.VersionHistoryResponse{
		ID:       id,
		Title:    title,
		Versions: versions,
	}, nil
}

// SetActiveParameterSet updates the active parameter set for a schema
func (s *Service) SetActiveParameterSet(ctx context.Context, id, parameterSet string) (*models.DetectionSchema, error) {
	// Get the latest version
	schema, err := s.repo.GetLatestSchemaByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SetActiveParameterSet(ctx, schema.VersionID, parameterSet); err != nil {
		return nil, fmt.Errorf("failed to set active parameter set: %w", err)
	}

	return s.repo.GetSchemaByVersionID(ctx, schema.VersionID)
}

// =============================================================================
// Case Methods
// =============================================================================

// CreateCase creates a new security case
func (s *Service) CreateCase(ctx context.Context, req *models.CreateCaseRequest, userID string) (*models.Case, error) {
	c := &models.Case{
		Title:       req.Title,
		Description: req.Description,
		Severity:    req.Severity,
		Priority:    req.Priority,
		Status:      models.CaseStatusOpen,
		AssigneeID:  req.AssigneeID,
		CreatedBy:   userID,
	}

	// Set defaults
	if c.Severity == "" {
		c.Severity = models.SeverityMedium
	}
	if c.Priority == "" {
		c.Priority = models.PriorityMedium
	}

	// Generate ID
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate case ID: %w", err)
	}
	c.ID = id.String()

	if err := s.repo.CreateCase(ctx, c); err != nil {
		return nil, fmt.Errorf("failed to create case: %w", err)
	}

	// Add alerts if provided
	if len(req.AlertIDs) > 0 {
		alerts := make([]*models.CaseAlert, len(req.AlertIDs))
		for i, alertID := range req.AlertIDs {
			alerts[i] = &models.CaseAlert{AlertID: alertID}
		}
		// Log but don't fail - case was created successfully
		if err := s.repo.AddAlertsToCase(ctx, c.ID, alerts, userID); err != nil {
			log.Printf("Warning: Failed to add alerts to case %s: %v (case_id=%s, alert_count=%d, user_id=%s)",
				c.ID, err, c.ID, len(alerts), userID)
		}
	}

	return s.repo.GetCaseByID(ctx, c.ID)
}

// GetCase retrieves a case by ID
func (s *Service) GetCase(ctx context.Context, id string) (*models.Case, error) {
	return s.repo.GetCaseByID(ctx, id)
}

// ListCases retrieves a paginated list of cases
func (s *Service) ListCases(ctx context.Context, req *models.ListCasesRequest) (*models.ListCasesResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	cases, total, err := s.repo.ListCases(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list cases: %w", err)
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

// UpdateCase updates a case
func (s *Service) UpdateCase(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) (*models.Case, error) {
	if err := s.repo.UpdateCase(ctx, id, req, userID); err != nil {
		return nil, fmt.Errorf("failed to update case: %w", err)
	}

	return s.repo.GetCaseByID(ctx, id)
}

// CloseCase closes a case
func (s *Service) CloseCase(ctx context.Context, id, userID string) (*models.Case, error) {
	if err := s.repo.CloseCase(ctx, id, userID); err != nil {
		return nil, fmt.Errorf("failed to close case: %w", err)
	}

	return s.repo.GetCaseByID(ctx, id)
}

// ReopenCase reopens a closed case
func (s *Service) ReopenCase(ctx context.Context, id string) (*models.Case, error) {
	if err := s.repo.ReopenCase(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to reopen case: %w", err)
	}

	return s.repo.GetCaseByID(ctx, id)
}

// AddAlertsToCase adds alerts to a case
func (s *Service) AddAlertsToCase(ctx context.Context, caseID string, alertIDs []string, userID string) error {
	alerts := make([]*models.CaseAlert, len(alertIDs))
	for i, alertID := range alertIDs {
		alerts[i] = &models.CaseAlert{AlertID: alertID}
	}

	return s.repo.AddAlertsToCase(ctx, caseID, alerts, userID)
}

// GetCaseAlerts retrieves all alerts for a case
func (s *Service) GetCaseAlerts(ctx context.Context, caseID string) ([]*models.CaseAlert, error) {
	return s.repo.GetCaseAlerts(ctx, caseID)
}

// RemoveAlertFromCase removes an alert from a case
func (s *Service) RemoveAlertFromCase(ctx context.Context, caseID, alertID string) error {
	return s.repo.RemoveAlertFromCase(ctx, caseID, alertID)
}
