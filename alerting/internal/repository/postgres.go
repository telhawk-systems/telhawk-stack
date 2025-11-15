package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(ctx context.Context, connString string) (*PostgresRepository, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Connection pool configuration
	config.MaxConns = 25            // Maximum number of connections in the pool
	config.MinConns = 5             // Minimum idle connections
	config.MaxConnLifetime = 5 * 60 // 5 minutes max connection lifetime
	config.MaxConnIdleTime = 1 * 60 // 1 minute max idle time

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

// CreateCase creates a new case
func (r *PostgresRepository) CreateCase(ctx context.Context, c *models.Case) error {
	query := `
		INSERT INTO cases (id, title, description, severity, status, assignee, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		c.ID, c.Title, c.Description, c.Severity, c.Status,
		c.Assignee, c.CreatedBy, c.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create case: %w", err)
	}

	return nil
}

// GetCaseByID retrieves a case by ID with alert count
func (r *PostgresRepository) GetCaseByID(ctx context.Context, id string) (*models.Case, error) {
	query := `
		SELECT
			c.id, c.title, c.description, c.severity, c.status,
			c.assignee, c.created_by, c.created_at, c.updated_at,
			c.closed_at, c.closed_by,
			COUNT(ca.alert_id) as alert_count
		FROM cases c
		LEFT JOIN case_alerts ca ON c.id = ca.case_id
		WHERE c.id = $1
		GROUP BY c.id
	`

	c := &models.Case{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Title, &c.Description, &c.Severity, &c.Status,
		&c.Assignee, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		&c.ClosedAt, &c.ClosedBy, &c.AlertCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCaseNotFound
		}
		return nil, fmt.Errorf("failed to get case: %w", err)
	}

	return c, nil
}

// ListCases retrieves a paginated list of cases
func (r *PostgresRepository) ListCases(ctx context.Context, req *models.ListCasesRequest) ([]*models.Case, int, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if req.Status != "" {
		whereClause += fmt.Sprintf(" AND c.status = $%d", argPos)
		args = append(args, req.Status)
		argPos++
	}
	if req.Severity != "" {
		whereClause += fmt.Sprintf(" AND c.severity = $%d", argPos)
		args = append(args, req.Severity)
		argPos++
	}
	if req.Assignee != "" {
		whereClause += fmt.Sprintf(" AND c.assignee = $%d", argPos)
		args = append(args, req.Assignee)
		argPos++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM cases c %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count cases: %w", err)
	}

	// Query cases with alert count
	offset := (req.Page - 1) * req.Limit
	args = append(args, req.Limit, offset)

	query := fmt.Sprintf(`
		SELECT
			c.id, c.title, c.description, c.severity, c.status,
			c.assignee, c.created_by, c.created_at, c.updated_at,
			c.closed_at, c.closed_by,
			COUNT(ca.alert_id) as alert_count
		FROM cases c
		LEFT JOIN case_alerts ca ON c.id = ca.case_id
		%s
		GROUP BY c.id
		ORDER BY c.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list cases: %w", err)
	}
	defer rows.Close()

	cases := []*models.Case{}
	for rows.Next() {
		c := &models.Case{}
		if err := rows.Scan(
			&c.ID, &c.Title, &c.Description, &c.Severity, &c.Status,
			&c.Assignee, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
			&c.ClosedAt, &c.ClosedBy, &c.AlertCount,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan case: %w", err)
		}
		cases = append(cases, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return cases, total, nil
}

// UpdateCase updates case fields
func (r *PostgresRepository) UpdateCase(ctx context.Context, id string, req *models.UpdateCaseRequest, userID string) error {
	// Build dynamic UPDATE query
	setClauses := []string{"updated_at = $1"}
	args := []interface{}{time.Now()}
	argPos := 2

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argPos))
		args = append(args, *req.Title)
		argPos++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *req.Description)
		argPos++
	}
	if req.Severity != nil {
		setClauses = append(setClauses, fmt.Sprintf("severity = $%d", argPos))
		args = append(args, *req.Severity)
		argPos++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *req.Status)
		argPos++
	}
	if req.Assignee != nil {
		setClauses = append(setClauses, fmt.Sprintf("assignee = $%d", argPos))
		args = append(args, *req.Assignee)
		argPos++
	}

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE cases
		SET %s
		WHERE id = $%d
	`, joinStrings(setClauses, ", "), argPos)

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update case: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCaseNotFound
	}

	return nil
}

// CloseCase closes a case
func (r *PostgresRepository) CloseCase(ctx context.Context, id string, userID string) error {
	query := `
		UPDATE cases
		SET status = 'closed', closed_at = $1, closed_by = $2, updated_at = $1
		WHERE id = $3
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, now, userID, id)
	if err != nil {
		return fmt.Errorf("failed to close case: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCaseNotFound
	}

	return nil
}

// ReopenCase reopens a closed case
func (r *PostgresRepository) ReopenCase(ctx context.Context, id string) error {
	query := `
		UPDATE cases
		SET status = 'open', closed_at = NULL, closed_by = NULL, updated_at = $1
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to reopen case: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCaseNotFound
	}

	return nil
}

// AddAlertsToCase adds alerts to a case
func (r *PostgresRepository) AddAlertsToCase(ctx context.Context, caseID string, alertIDs []string, userID string) error {
	// First verify case exists
	var exists bool
	if err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM cases WHERE id = $1)", caseID).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check case existence: %w", err)
	}
	if !exists {
		return ErrCaseNotFound
	}

	// TODO: Fetch detection schema IDs from OpenSearch alerts
	// For now, we'll use placeholder values
	// In production, this would query OpenSearch to get the detection_schema_id and detection_schema_version_id

	now := time.Now()
	for _, alertID := range alertIDs {
		query := `
			INSERT INTO case_alerts (case_id, alert_id, detection_schema_id, detection_schema_version_id, added_at, added_by)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (case_id, alert_id) DO NOTHING
		`

		// TODO: Replace with actual schema IDs from OpenSearch
		placeholderSchemaID := "00000000-0000-0000-0000-000000000000"
		placeholderVersionID := "00000000-0000-0000-0000-000000000000"

		_, err := r.pool.Exec(ctx, query, caseID, alertID, placeholderSchemaID, placeholderVersionID, now, userID)
		if err != nil {
			return fmt.Errorf("failed to add alert to case: %w", err)
		}
	}

	return nil
}

// GetCaseAlerts retrieves all alerts for a case
func (r *PostgresRepository) GetCaseAlerts(ctx context.Context, caseID string) ([]*models.CaseAlert, error) {
	query := `
		SELECT case_id, alert_id, detection_schema_id, detection_schema_version_id, added_at, added_by
		FROM case_alerts
		WHERE case_id = $1
		ORDER BY added_at DESC
	`

	rows, err := r.pool.Query(ctx, query, caseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get case alerts: %w", err)
	}
	defer rows.Close()

	alerts := []*models.CaseAlert{}
	for rows.Next() {
		a := &models.CaseAlert{}
		if err := rows.Scan(&a.CaseID, &a.AlertID, &a.DetectionSchemaID, &a.DetectionSchemaVersionID, &a.AddedAt, &a.AddedBy); err != nil {
			return nil, fmt.Errorf("failed to scan case alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return alerts, nil
}

// Close closes the database connection pool
func (r *PostgresRepository) Close() error {
	r.pool.Close()
	return nil
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
