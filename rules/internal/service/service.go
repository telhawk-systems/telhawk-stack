package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/models"
	"github.com/telhawk-systems/telhawk-stack/rules/internal/repository"
)

type Service struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

// CreateSchema creates a new detection schema (generates new id and version_id)
func (s *Service) CreateSchema(ctx context.Context, req *models.CreateSchemaRequest, userID string) (*models.DetectionSchema, error) {
	idUUID, _ := uuid.NewV7()
	versionUUID, _ := uuid.NewV7()
	schema := &models.DetectionSchema{
		ID:         idUUID.String(),      // Server-generated stable ID
		VersionID:  versionUUID.String(), // Server-generated version ID
		Model:      req.Model,
		View:       req.View,
		Controller: req.Controller,
		CreatedBy:  userID,
	}

	if err := s.repo.CreateSchema(ctx, schema); err != nil {
		return nil, err
	}

	// Retrieve the created schema with calculated version number
	return s.repo.GetSchemaByVersionID(ctx, schema.VersionID)
}

// UpdateSchema creates a new version of an existing detection schema
func (s *Service) UpdateSchema(ctx context.Context, id string, req *models.UpdateSchemaRequest, userID string) (*models.DetectionSchema, error) {
	// Verify the rule exists
	_, err := s.repo.GetLatestSchemaByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %w", err)
	}

	versionUUID, _ := uuid.NewV7()
	schema := &models.DetectionSchema{
		ID:         id,                   // Same stable ID
		VersionID:  versionUUID.String(), // New version ID
		Model:      req.Model,
		View:       req.View,
		Controller: req.Controller,
		CreatedBy:  userID,
	}

	if err := s.repo.CreateSchema(ctx, schema); err != nil {
		return nil, err
	}

	// Retrieve the created schema with calculated version number
	return s.repo.GetSchemaByVersionID(ctx, schema.VersionID)
}

// GetSchema retrieves a detection schema by version ID or stable ID (latest version)
func (s *Service) GetSchema(ctx context.Context, idOrVersionID string, version *int) (*models.DetectionSchema, error) {
	// Try as version_id first
	schema, err := s.repo.GetSchemaByVersionID(ctx, idOrVersionID)
	if err == nil {
		return schema, nil
	}

	// If not found, try as stable ID (get latest version)
	if err == repository.ErrSchemaNotFound {
		schema, err = s.repo.GetLatestSchemaByID(ctx, idOrVersionID)
		if err != nil {
			return nil, err
		}

		// If specific version requested, get it
		if version != nil {
			// TODO: Implement get by stable ID + version number
			// For now, just return latest
		}

		return schema, nil
	}

	return nil, err
}

// ListSchemas retrieves a paginated list of detection schemas
func (s *Service) ListSchemas(ctx context.Context, req *models.ListSchemasRequest) (*models.ListSchemasResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 50
	}

	schemas, total, err := s.repo.ListSchemas(ctx, req)
	if err != nil {
		return nil, err
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

// GetVersionHistory retrieves all versions of a detection schema
func (s *Service) GetVersionHistory(ctx context.Context, id string) (*models.VersionHistoryResponse, error) {
	versions, err := s.repo.GetSchemaVersionHistory(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, repository.ErrSchemaNotFound
	}

	return &models.VersionHistoryResponse{
		ID:       id,
		Title:    versions[0].Title, // Latest version title
		Versions: versions,
	}, nil
}

// DisableSchema disables a specific version
func (s *Service) DisableSchema(ctx context.Context, versionID, userID string) error {
	return s.repo.DisableSchema(ctx, versionID, userID)
}

// EnableSchema re-enables a disabled version
func (s *Service) EnableSchema(ctx context.Context, versionID string) error {
	return s.repo.EnableSchema(ctx, versionID)
}

// HideSchema hides (soft deletes) a specific version
func (s *Service) HideSchema(ctx context.Context, versionID, userID string) error {
	return s.repo.HideSchema(ctx, versionID, userID)
}
