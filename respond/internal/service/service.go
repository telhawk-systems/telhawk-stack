package service

import (
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

// TODO: Add rules service methods when migrating from rules service
// - CreateSchema
// - GetSchema
// - ListSchemas
// - UpdateSchema
// - DisableSchema
// - EnableSchema
// - HideSchema
// - GetVersionHistory
// - ValidateCorrelation

// TODO: Add alerting service methods when migrating from alerting service
// - CreateCase
// - GetCase
// - ListCases
// - UpdateCase
// - CloseCase
// - ReopenCase
// - AddAlertsToCase
// - GetCaseAlerts
