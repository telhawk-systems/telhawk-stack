package repository

import (
	"context"
	"errors"

	"github.com/telhawk-systems/telhawk-stack/alerting/internal/models"
)

var (
	ErrCaseNotFound = errors.New("case not found")
	ErrAlertExists  = errors.New("alert already exists in case")
)

// Repository defines the interface for case and alert persistence
type Repository interface {
	// Case operations
	CreateCase(ctx context.Context, c *models.Case) error
	GetCaseByID(ctx context.Context, id string) (*models.Case, error)
	ListCases(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error)
	UpdateCase(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) error
	CloseCase(ctx context.Context, id string, userID string) error
	ReopenCase(ctx context.Context, id string) error

	// Case-Alert operations
	AddAlertsToCase(ctx context.Context, caseID string, alertIDs []string, userID string) error
	GetCaseAlerts(ctx context.Context, caseID string) ([]*models.CaseAlert, error)

	// Utility
	Close() error
}
