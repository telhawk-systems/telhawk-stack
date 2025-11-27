package database

import (
	"context"
	"time"
)

// Standard timeout durations for database operations
const (
	// DefaultQueryTimeout is the timeout for read queries
	DefaultQueryTimeout = 5 * time.Second

	// DefaultWriteTimeout is the timeout for write operations
	DefaultWriteTimeout = 10 * time.Second

	// DefaultBulkTimeout is the timeout for bulk operations
	DefaultBulkTimeout = 30 * time.Second
)

// QueryContext creates a context with DefaultQueryTimeout.
// Use this for SELECT queries and read operations.
func QueryContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DefaultQueryTimeout)
}

// WriteContext creates a context with DefaultWriteTimeout.
// Use this for INSERT, UPDATE, DELETE operations.
func WriteContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DefaultWriteTimeout)
}

// BulkContext creates a context with DefaultBulkTimeout.
// Use this for bulk operations and migrations.
func BulkContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DefaultBulkTimeout)
}
