package repository

import (
	"context"
	"errors"

	"github.com/telhawk-systems/telhawk-stack/respond/internal/models"
)

var (
	// ErrSchemaNotFound is returned when a detection schema is not found.
	ErrSchemaNotFound = errors.New("detection schema not found")
	// ErrSchemaExists is returned when a detection schema already exists.
	ErrSchemaExists = errors.New("detection schema already exists")

	// ErrCaseNotFound is returned when a case is not found.
	ErrCaseNotFound = errors.New("case not found")
	// ErrAlertExists is returned when an alert already exists in a case.
	ErrAlertExists = errors.New("alert already exists in case")
)

// Repository defines the interface for respond data storage
type Repository interface {
	// Health check
	Ping(ctx context.Context) error
	Close() error

	// Detection Schema operations
	CreateSchema(ctx context.Context, schema *models.DetectionSchema) error
	GetSchemaByVersionID(ctx context.Context, versionID string) (*models.DetectionSchema, error)
	GetLatestSchemaByID(ctx context.Context, id string) (*models.DetectionSchema, error)
	GetSchemaVersionHistory(ctx context.Context, id string) ([]*models.DetectionSchemaVersion, error)
	ListSchemas(ctx context.Context, req *models.ListSchemasRequest) ([]*models.DetectionSchema, int, error)
	DisableSchema(ctx context.Context, versionID, userID string) error
	EnableSchema(ctx context.Context, versionID string) error
	HideSchema(ctx context.Context, versionID, userID string) error
	SetActiveParameterSet(ctx context.Context, versionID, parameterSet string) error

	// Case operations
	CreateCase(ctx context.Context, c *models.Case) error
	GetCaseByID(ctx context.Context, id string) (*models.Case, error)
	ListCases(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error)
	UpdateCase(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) error
	CloseCase(ctx context.Context, id string, userID string) error
	ReopenCase(ctx context.Context, id string) error

	// Case-Alert operations
	AddAlertsToCase(ctx context.Context, caseID string, alerts []*models.CaseAlert, userID string) error
	GetCaseAlerts(ctx context.Context, caseID string) ([]*models.CaseAlert, error)
	RemoveAlertFromCase(ctx context.Context, caseID, alertID string) error
}
