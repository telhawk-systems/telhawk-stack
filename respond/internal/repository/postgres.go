// Package repository provides data storage implementations for the respond service.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/models"
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
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 5 * time.Minute
	config.MaxConnIdleTime = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

// Ping checks database connectivity
func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Close closes the database connection pool
func (r *PostgresRepository) Close() error {
	r.pool.Close()
	return nil
}

// =============================================================================
// Detection Schema Methods
// =============================================================================

// CreateSchema creates a new detection schema version
func (r *PostgresRepository) CreateSchema(ctx context.Context, schema *models.DetectionSchema) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Generate version_id if not provided
	if schema.VersionID == "" {
		versionUUID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to generate version UUID: %w", err)
		}
		schema.VersionID = versionUUID.String()
	}

	// Generate id if not provided (first version)
	if schema.ID == "" {
		idUUID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to generate ID UUID: %w", err)
		}
		schema.ID = idUUID.String()
	}

	// Marshal JSONB fields
	modelJSON, err := json.Marshal(schema.Model)
	if err != nil {
		return fmt.Errorf("failed to marshal model: %w", err)
	}

	viewJSON, err := json.Marshal(schema.View)
	if err != nil {
		return fmt.Errorf("failed to marshal view: %w", err)
	}

	controllerJSON, err := json.Marshal(schema.Controller)
	if err != nil {
		return fmt.Errorf("failed to marshal controller: %w", err)
	}

	query := `
		INSERT INTO detection_schemas
		(id, version_id, model, view, controller, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`

	_, err = r.pool.Exec(ctx, query,
		schema.ID,
		schema.VersionID,
		modelJSON,
		viewJSON,
		controllerJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create detection schema: %w", err)
	}

	return nil
}

// GetSchemaByVersionID retrieves a specific version of a detection schema
func (r *PostgresRepository) GetSchemaByVersionID(ctx context.Context, versionID string) (*models.DetectionSchema, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT
			id, version_id, model, view, controller, created_at,
			disabled_at, disabled_by, hidden_at, hidden_by,
			ROW_NUMBER() OVER (PARTITION BY id ORDER BY created_at) as version
		FROM detection_schemas
		WHERE version_id = $1
	`

	var schema models.DetectionSchema
	var modelJSON, viewJSON, controllerJSON []byte

	err := r.pool.QueryRow(ctx, query, versionID).Scan(
		&schema.ID,
		&schema.VersionID,
		&modelJSON,
		&viewJSON,
		&controllerJSON,
		&schema.CreatedAt,
		&schema.DisabledAt,
		&schema.DisabledBy,
		&schema.HiddenAt,
		&schema.HiddenBy,
		&schema.Version,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSchemaNotFound
		}
		return nil, fmt.Errorf("failed to get detection schema: %w", err)
	}

	// Unmarshal JSONB fields
	if err := json.Unmarshal(modelJSON, &schema.Model); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model: %w", err)
	}
	if err := json.Unmarshal(viewJSON, &schema.View); err != nil {
		return nil, fmt.Errorf("failed to unmarshal view: %w", err)
	}
	if err := json.Unmarshal(controllerJSON, &schema.Controller); err != nil {
		return nil, fmt.Errorf("failed to unmarshal controller: %w", err)
	}

	return &schema, nil
}

// GetLatestSchemaByID retrieves the latest version of a detection schema by stable ID
func (r *PostgresRepository) GetLatestSchemaByID(ctx context.Context, id string) (*models.DetectionSchema, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		WITH ranked AS (
			SELECT
				id, version_id, model, view, controller, created_at,
				disabled_at, disabled_by, hidden_at, hidden_by,
				ROW_NUMBER() OVER (PARTITION BY id ORDER BY created_at DESC) as rn,
				COUNT(*) OVER (PARTITION BY id) as version
			FROM detection_schemas
			WHERE id = $1
		)
		SELECT id, version_id, model, view, controller, created_at,
			disabled_at, disabled_by, hidden_at, hidden_by, version
		FROM ranked
		WHERE rn = 1
	`

	var schema models.DetectionSchema
	var modelJSON, viewJSON, controllerJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&schema.ID,
		&schema.VersionID,
		&modelJSON,
		&viewJSON,
		&controllerJSON,
		&schema.CreatedAt,
		&schema.DisabledAt,
		&schema.DisabledBy,
		&schema.HiddenAt,
		&schema.HiddenBy,
		&schema.Version,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSchemaNotFound
		}
		return nil, fmt.Errorf("failed to get detection schema: %w", err)
	}

	// Unmarshal JSONB fields
	if err := json.Unmarshal(modelJSON, &schema.Model); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model: %w", err)
	}
	if err := json.Unmarshal(viewJSON, &schema.View); err != nil {
		return nil, fmt.Errorf("failed to unmarshal view: %w", err)
	}
	if err := json.Unmarshal(controllerJSON, &schema.Controller); err != nil {
		return nil, fmt.Errorf("failed to unmarshal controller: %w", err)
	}

	return &schema, nil
}

// GetSchemaVersionHistory retrieves all versions of a detection schema
func (r *PostgresRepository) GetSchemaVersionHistory(ctx context.Context, id string) ([]*models.DetectionSchemaVersion, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT
			version_id,
			ROW_NUMBER() OVER (PARTITION BY id ORDER BY created_at) as version,
			view->>'title' as title,
			created_at,
			disabled_at
		FROM detection_schemas
		WHERE id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get version history: %w", err)
	}
	defer rows.Close()

	var versions []*models.DetectionSchemaVersion
	for rows.Next() {
		var v models.DetectionSchemaVersion
		var title *string
		err := rows.Scan(
			&v.VersionID,
			&v.Version,
			&title,
			&v.CreatedAt,
			&v.DisabledAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		if title != nil {
			v.Title = *title
		}
		versions = append(versions, &v)
	}

	if len(versions) == 0 {
		return nil, ErrSchemaNotFound
	}

	return versions, nil
}

// ListSchemas retrieves a paginated list of detection schemas
func (r *PostgresRepository) ListSchemas(ctx context.Context, req *models.ListSchemasRequest) ([]*models.DetectionSchema, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Build WHERE clause
	where := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if !req.IncludeDisabled {
		where += " AND disabled_at IS NULL"
	}

	if !req.IncludeHidden {
		where += " AND hidden_at IS NULL"
	}

	if req.ID != "" {
		argCount++
		where += fmt.Sprintf(" AND id = $%d", argCount)
		args = append(args, req.ID)
	}

	if req.Severity != "" {
		argCount++
		where += fmt.Sprintf(" AND view->>'severity' = $%d", argCount)
		args = append(args, req.Severity)
	}

	if req.Title != "" {
		argCount++
		where += fmt.Sprintf(" AND view->>'title' ILIKE $%d", argCount)
		args = append(args, "%"+req.Title+"%")
	}

	// Get total count
	countQuery := `SELECT COUNT(DISTINCT id) FROM detection_schemas ` + where
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count schemas: %w", err)
	}

	// Get latest version of each schema
	query := fmt.Sprintf(`
		WITH ranked_schemas AS (
			SELECT
				id, version_id, model, view, controller, created_at,
				disabled_at, disabled_by, hidden_at, hidden_by,
				ROW_NUMBER() OVER (PARTITION BY id ORDER BY created_at DESC) as rn,
				COUNT(*) OVER (PARTITION BY id) as version
			FROM detection_schemas
			%s
		)
		SELECT
			id, version_id, model, view, controller, created_at,
			disabled_at, disabled_by, hidden_at, hidden_by, version
		FROM ranked_schemas
		WHERE rn = 1
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argCount+1, argCount+2)

	args = append(args, req.Limit, (req.Page-1)*req.Limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []*models.DetectionSchema
	for rows.Next() {
		var schema models.DetectionSchema
		var modelJSON, viewJSON, controllerJSON []byte

		err := rows.Scan(
			&schema.ID,
			&schema.VersionID,
			&modelJSON,
			&viewJSON,
			&controllerJSON,
			&schema.CreatedAt,
			&schema.DisabledAt,
			&schema.DisabledBy,
			&schema.HiddenAt,
			&schema.HiddenBy,
			&schema.Version,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan schema: %w", err)
		}

		// Unmarshal JSONB fields
		if err := json.Unmarshal(modelJSON, &schema.Model); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal model: %w", err)
		}
		if err := json.Unmarshal(viewJSON, &schema.View); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal view: %w", err)
		}
		if err := json.Unmarshal(controllerJSON, &schema.Controller); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal controller: %w", err)
		}

		schemas = append(schemas, &schema)
	}

	return schemas, total, nil
}

// DisableSchema disables a specific version
func (r *PostgresRepository) DisableSchema(ctx context.Context, versionID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE detection_schemas
		SET disabled_at = NOW(), disabled_by = $2
		WHERE version_id = $1 AND disabled_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, versionID, userID)
	if err != nil {
		return fmt.Errorf("failed to disable schema: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSchemaNotFound
	}

	return nil
}

// EnableSchema re-enables a disabled version
func (r *PostgresRepository) EnableSchema(ctx context.Context, versionID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE detection_schemas
		SET disabled_at = NULL, disabled_by = NULL
		WHERE version_id = $1
	`

	result, err := r.pool.Exec(ctx, query, versionID)
	if err != nil {
		return fmt.Errorf("failed to enable schema: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSchemaNotFound
	}

	return nil
}

// HideSchema hides (soft deletes) a specific version
func (r *PostgresRepository) HideSchema(ctx context.Context, versionID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE detection_schemas
		SET hidden_at = NOW(), hidden_by = $2
		WHERE version_id = $1 AND hidden_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, versionID, userID)
	if err != nil {
		return fmt.Errorf("failed to hide schema: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSchemaNotFound
	}

	return nil
}

// SetActiveParameterSet updates the active parameter set for a schema version
func (r *PostgresRepository) SetActiveParameterSet(ctx context.Context, versionID, parameterSet string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE detection_schemas
		SET active_parameter_set = $2
		WHERE version_id = $1
		AND disabled_at IS NULL
		AND hidden_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, versionID, parameterSet)
	if err != nil {
		return fmt.Errorf("failed to set active parameter set: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSchemaNotFound
	}

	return nil
}

// =============================================================================
// Case Methods
// =============================================================================

// CreateCase creates a new case
func (r *PostgresRepository) CreateCase(ctx context.Context, c *models.Case) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Generate ID if not provided
	if c.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to generate case UUID: %w", err)
		}
		c.ID = id.String()
	}

	query := `
		INSERT INTO cases (id, title, description, status, priority, assignee_id,
			detection_schema_id, detection_schema_version_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`

	_, err := r.pool.Exec(ctx, query,
		c.ID, c.Title, c.Description, c.Status, c.Priority,
		c.AssigneeID, c.DetectionSchemaID, c.DetectionSchemaVersionID,
	)
	if err != nil {
		return fmt.Errorf("failed to create case: %w", err)
	}

	return nil
}

// GetCaseByID retrieves a case by ID with alert count
func (r *PostgresRepository) GetCaseByID(ctx context.Context, id string) (*models.Case, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT
			c.id, c.title, c.description, c.status, c.priority,
			c.assignee_id, c.detection_schema_id, c.detection_schema_version_id,
			c.created_at, c.updated_at, c.closed_at, c.closed_by,
			COUNT(ca.alert_id) as alert_count
		FROM cases c
		LEFT JOIN case_alerts ca ON c.id = ca.case_id
		WHERE c.id = $1
		GROUP BY c.id
	`

	c := &models.Case{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Title, &c.Description, &c.Status, &c.Priority,
		&c.AssigneeID, &c.DetectionSchemaID, &c.DetectionSchemaVersionID,
		&c.CreatedAt, &c.UpdatedAt, &c.ClosedAt, &c.ClosedBy, &c.AlertCount,
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if req.Status != "" {
		whereClause += fmt.Sprintf(" AND c.status = $%d", argPos)
		args = append(args, req.Status)
		argPos++
	}
	if req.Priority != "" {
		whereClause += fmt.Sprintf(" AND c.priority = $%d", argPos)
		args = append(args, req.Priority)
		argPos++
	}
	if req.AssigneeID != "" {
		whereClause += fmt.Sprintf(" AND c.assignee_id = $%d", argPos)
		args = append(args, req.AssigneeID)
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
			c.id, c.title, c.description, c.status, c.priority,
			c.assignee_id, c.detection_schema_id, c.detection_schema_version_id,
			c.created_at, c.updated_at, c.closed_at, c.closed_by,
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
			&c.ID, &c.Title, &c.Description, &c.Status, &c.Priority,
			&c.AssigneeID, &c.DetectionSchemaID, &c.DetectionSchemaVersionID,
			&c.CreatedAt, &c.UpdatedAt, &c.ClosedAt, &c.ClosedBy, &c.AlertCount,
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Build dynamic UPDATE query
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argPos := 1

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
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *req.Status)
		argPos++
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argPos))
		args = append(args, *req.Priority)
		argPos++
	}
	if req.AssigneeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", argPos))
		args = append(args, *req.AssigneeID)
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
func (r *PostgresRepository) CloseCase(ctx context.Context, id, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE cases
		SET status = 'closed', closed_at = NOW(), closed_by = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, userID)
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE cases
		SET status = 'open', closed_at = NULL, closed_by = NULL, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to reopen case: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCaseNotFound
	}

	return nil
}

// =============================================================================
// Case-Alert Methods
// =============================================================================

// AddAlertsToCase adds alerts to a case
func (r *PostgresRepository) AddAlertsToCase(ctx context.Context, caseID string, alerts []*models.CaseAlert, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// First verify case exists
	var exists bool
	if err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM cases WHERE id = $1)", caseID).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check case existence: %w", err)
	}
	if !exists {
		return ErrCaseNotFound
	}

	for _, alert := range alerts {
		// Generate ID if not provided
		if alert.ID == "" {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("failed to generate alert UUID: %w", err)
			}
			alert.ID = id.String()
		}

		query := `
			INSERT INTO case_alerts (id, case_id, alert_id, added_at, added_by)
			VALUES ($1, $2, $3, NOW(), $4)
			ON CONFLICT (case_id, alert_id) DO NOTHING
		`

		_, err := r.pool.Exec(ctx, query, alert.ID, caseID, alert.AlertID, userID)
		if err != nil {
			return fmt.Errorf("failed to add alert to case: %w", err)
		}
	}

	return nil
}

// GetCaseAlerts retrieves all alerts for a case
func (r *PostgresRepository) GetCaseAlerts(ctx context.Context, caseID string) ([]*models.CaseAlert, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, case_id, alert_id, added_at, added_by
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
		if err := rows.Scan(&a.ID, &a.CaseID, &a.AlertID, &a.AddedAt, &a.AddedBy); err != nil {
			return nil, fmt.Errorf("failed to scan case alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return alerts, nil
}

// RemoveAlertFromCase removes an alert from a case
func (r *PostgresRepository) RemoveAlertFromCase(ctx context.Context, caseID, alertID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `DELETE FROM case_alerts WHERE case_id = $1 AND alert_id = $2`

	result, err := r.pool.Exec(ctx, query, caseID, alertID)
	if err != nil {
		return fmt.Errorf("failed to remove alert from case: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("alert not found in case")
	}

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// joinStrings joins strings with a separator
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
