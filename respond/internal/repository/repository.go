package repository

import (
	"context"
)

// Repository defines the interface for respond data storage
type Repository interface {
	// Health check
	Ping(ctx context.Context) error
	Close() error

	// TODO: Add rules repository methods when migrating
	// - CreateSchema
	// - GetSchema
	// - GetSchemaByVersionID
	// - ListSchemas
	// - UpdateSchema
	// - DisableSchema
	// - EnableSchema
	// - HideSchema
	// - GetVersionHistory

	// TODO: Add alerting repository methods when migrating
	// - CreateCase
	// - GetCase
	// - ListCases
	// - UpdateCase
	// - CloseCase
	// - ReopenCase
	// - AddAlertsToCase
	// - GetCaseAlerts
	// - RemoveAlertFromCase
}
