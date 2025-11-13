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
	"github.com/telhawk-systems/telhawk-stack/rules/internal/models"
)

var (
	ErrSchemaNotFound = errors.New("detection schema not found")
	ErrSchemaExists   = errors.New("detection schema already exists")
)

type Repository interface {
	CreateSchema(ctx context.Context, schema *models.DetectionSchema) error
	GetSchemaByVersionID(ctx context.Context, versionID string) (*models.DetectionSchema, error)
	GetLatestSchemaByID(ctx context.Context, id string) (*models.DetectionSchema, error)
	GetSchemaVersionHistory(ctx context.Context, id string) ([]*models.DetectionSchemaVersion, error)
	ListSchemas(ctx context.Context, req *models.ListSchemasRequest) ([]*models.DetectionSchema, int, error)
	DisableSchema(ctx context.Context, versionID, userID string) error
	EnableSchema(ctx context.Context, versionID string) error
	HideSchema(ctx context.Context, versionID, userID string) error
	SetActiveParameterSet(ctx context.Context, versionID, parameterSet string) error
	Close()
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

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

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

func (r *PostgresRepository) Close() {
	r.pool.Close()
}

// CreateSchema creates a new detection schema version
func (r *PostgresRepository) CreateSchema(ctx context.Context, schema *models.DetectionSchema) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Generate version_id if not provided
	if schema.VersionID == "" {
		versionUUID, _ := uuid.NewV7()
		schema.VersionID = versionUUID.String()
	}

	// Generate id if not provided (first version)
	if schema.ID == "" {
		idUUID, _ := uuid.NewV7()
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
		(id, version_id, model, view, controller, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	_, err = r.pool.Exec(ctx, query,
		schema.ID,
		schema.VersionID,
		modelJSON,
		viewJSON,
		controllerJSON,
		schema.CreatedBy,
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
			id, version_id, model, view, controller, created_by, created_at,
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
		&schema.CreatedBy,
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
		SELECT
			id, version_id, model, view, controller, created_by, created_at,
			disabled_at, disabled_by, hidden_at, hidden_by,
			ROW_NUMBER() OVER (PARTITION BY id ORDER BY created_at) as version
		FROM detection_schemas
		WHERE id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var schema models.DetectionSchema
	var modelJSON, viewJSON, controllerJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&schema.ID,
		&schema.VersionID,
		&modelJSON,
		&viewJSON,
		&controllerJSON,
		&schema.CreatedBy,
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
			created_by,
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
		err := rows.Scan(
			&v.VersionID,
			&v.Version,
			&v.Title,
			&v.CreatedBy,
			&v.CreatedAt,
			&v.DisabledAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
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
	query := `
		WITH ranked_schemas AS (
			SELECT
				id, version_id, model, view, controller, created_by, created_at,
				disabled_at, disabled_by, hidden_at, hidden_by,
				ROW_NUMBER() OVER (PARTITION BY id ORDER BY created_at DESC) as rn,
				ROW_NUMBER() OVER (PARTITION BY id ORDER BY created_at) as version
			FROM detection_schemas
			` + where + `
		)
		SELECT
			id, version_id, model, view, controller, created_by, created_at,
			disabled_at, disabled_by, hidden_at, hidden_by, version
		FROM ranked_schemas
		WHERE rn = 1
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)

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
			&schema.CreatedBy,
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
// This is a tuning parameter change and does NOT create a new version
func (r *PostgresRepository) SetActiveParameterSet(ctx context.Context, versionID, parameterSet string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Update the model JSONB field to set active_parameter_set
	// Uses PostgreSQL's jsonb_set to update a specific key
	query := `
		UPDATE detection_schemas
		SET model = jsonb_set(
			COALESCE(model, '{}'::jsonb),
			'{active_parameter_set}',
			to_jsonb($2::text),
			true
		)
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
