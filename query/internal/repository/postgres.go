package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(ctx context.Context, connString string) (*PostgresRepository, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &PostgresRepository{pool: pool}, nil
}

func (r *PostgresRepository) Close() { r.pool.Close() }

// InsertVersion inserts a new saved_searches version row (immutable insert-only).
func (r *PostgresRepository) InsertVersion(ctx context.Context, s *models.SavedSearch) error {
	q := `INSERT INTO saved_searches (
            id, version_id, owner_id, created_by, name, query, filters, is_global,
            created_at, disabled_at, disabled_by, hidden_at, hidden_by
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`

	var owner interface{}
	if s.OwnerID != nil {
		owner = *s.OwnerID
	} else {
		owner = nil
	}
	var disabledAt interface{}
	var disabledBy interface{}
	if s.DisabledAt != nil {
		disabledAt = *s.DisabledAt
		disabledBy = s.CreatedBy
	} else {
		disabledAt = nil
		disabledBy = nil
	}
	var hiddenAt interface{}
	var hiddenBy interface{}
	if s.HiddenAt != nil {
		hiddenAt = *s.HiddenAt
		hiddenBy = s.CreatedBy
	} else {
		hiddenAt = nil
		hiddenBy = nil
	}

	_, err := r.pool.Exec(ctx, q,
		s.ID, s.VersionID, owner, s.CreatedBy, s.Name, s.Query, s.Filters, s.IsGlobal,
		s.CreatedAt, disabledAt, disabledBy, hiddenAt, hiddenBy,
	)
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	return nil
}

// GetLatest returns the latest version by created_at DESC, version_id DESC for a given id.
func (r *PostgresRepository) GetLatest(ctx context.Context, id string) (*models.SavedSearch, error) {
	q := `SELECT id, version_id, owner_id, created_by, name, query, filters, is_global,
                 created_at, disabled_at, hidden_at
          FROM saved_searches
          WHERE id = $1
          ORDER BY created_at DESC, version_id DESC
          LIMIT 1`
	var s models.SavedSearch
	var owner *string
	var disabledAt, hiddenAt *time.Time
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.VersionID, &owner, &s.CreatedBy, &s.Name, &s.Query, &s.Filters, &s.IsGlobal,
		&s.CreatedAt, &disabledAt, &hiddenAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("not found")
		}
		return nil, fmt.Errorf("get latest: %w", err)
	}
	s.OwnerID = owner
	s.DisabledAt = disabledAt
	s.HiddenAt = hiddenAt
	return &s, nil
}

// ListLatest returns the latest version per id; by default excludes hidden.
func (r *PostgresRepository) ListLatest(ctx context.Context, showAll bool) ([]models.SavedSearch, error) {
	base := `WITH latest AS (
                SELECT DISTINCT ON (id)
                    id, version_id, owner_id, created_by, name, query, filters, is_global,
                    created_at, disabled_at, hidden_at
                FROM saved_searches
                ORDER BY id, created_at DESC, version_id DESC
            )
            SELECT id, version_id, owner_id, created_by, name, query, filters, is_global,
                   created_at, disabled_at, hidden_at
            FROM latest`
	if !showAll {
		base += " WHERE hidden_at IS NULL"
	}
	// active-first (disabled_at NULL), then recency
	base += " ORDER BY (disabled_at IS NOT NULL), created_at DESC"

	rows, err := r.pool.Query(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("list latest: %w", err)
	}
	defer rows.Close()

	var out []models.SavedSearch
	for rows.Next() {
		var s models.SavedSearch
		var owner *string
		var disabledAt, hiddenAt *time.Time
		if err := rows.Scan(
			&s.ID, &s.VersionID, &owner, &s.CreatedBy, &s.Name, &s.Query, &s.Filters, &s.IsGlobal,
			&s.CreatedAt, &disabledAt, &hiddenAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		s.OwnerID = owner
		s.DisabledAt = disabledAt
		s.HiddenAt = hiddenAt
		out = append(out, s)
	}
	return out, nil
}

// ListLatestPaged returns paginated latest versions per id with total count.
func (r *PostgresRepository) ListLatestPaged(ctx context.Context, showAll bool, page, size int) ([]models.SavedSearch, int, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size

	baseLatest := `SELECT DISTINCT ON (id)
                    id, version_id, owner_id, created_by, name, query, filters, is_global,
                    created_at, disabled_at, hidden_at
                  FROM saved_searches
                  ORDER BY id, created_at DESC, version_id DESC`
	where := ""
	if !showAll {
		where = " WHERE hidden_at IS NULL"
	}

	// total count
	countSQL := "WITH latest AS (" + baseLatest + ") SELECT COUNT(*) FROM latest" + where
	var total int
	if err := r.pool.QueryRow(ctx, countSQL).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}

	// page query
	pageSQL := "WITH latest AS (" + baseLatest + ") SELECT id, version_id, owner_id, created_by, name, query, filters, is_global, created_at, disabled_at, hidden_at FROM latest" + where + " ORDER BY (disabled_at IS NOT NULL), created_at DESC LIMIT $1 OFFSET $2"
	rows, err := r.pool.Query(ctx, pageSQL, size, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list paged: %w", err)
	}
	defer rows.Close()

	var out []models.SavedSearch
	for rows.Next() {
		var s models.SavedSearch
		var owner *string
		var disabledAt, hiddenAt *time.Time
		if err := rows.Scan(
			&s.ID, &s.VersionID, &owner, &s.CreatedBy, &s.Name, &s.Query, &s.Filters, &s.IsGlobal,
			&s.CreatedAt, &disabledAt, &hiddenAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan: %w", err)
		}
		s.OwnerID = owner
		s.DisabledAt = disabledAt
		s.HiddenAt = hiddenAt
		out = append(out, s)
	}
	return out, total, nil
}

// ListLatestAfter returns the latest versions after a given cursor (created_at, version_id) for cursor pagination.
// Pass zero time and empty versionID to start from the top (most recent).
func (r *PostgresRepository) ListLatestAfter(ctx context.Context, showAll bool, cursorCreatedAt *time.Time, cursorVersionID string, size int) ([]models.SavedSearch, error) {
	if size <= 0 {
		size = 20
	}

	baseLatest := `SELECT DISTINCT ON (id)
                    id, version_id, owner_id, created_by, name, query, filters, is_global,
                    created_at, disabled_at, hidden_at
                  FROM saved_searches
                  ORDER BY id, created_at DESC, version_id DESC`
	where := ""
	args := []interface{}{}
	if !showAll {
		where = " WHERE hidden_at IS NULL"
	}
	// Post-filter using created_at/version_id for cursor
	if cursorCreatedAt != nil && !cursorCreatedAt.IsZero() && cursorVersionID != "" {
		if where == "" {
			where = " WHERE "
		} else {
			where += " AND "
		}
		// newer first ordering; to get the next page, we want items with (created_at, version_id) < cursor
		where += "(created_at, version_id) < ($1, $2)"
		args = append(args, *cursorCreatedAt, cursorVersionID)
	}
	sql := "WITH latest AS (" + baseLatest + ") SELECT id, version_id, owner_id, created_by, name, query, filters, is_global, created_at, disabled_at, hidden_at FROM latest" + where + " ORDER BY (disabled_at IS NOT NULL), created_at DESC, version_id DESC LIMIT $3"
	args = append(args, size)

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list after: %w", err)
	}
	defer rows.Close()

	var out []models.SavedSearch
	for rows.Next() {
		var s models.SavedSearch
		var owner *string
		var disabledAt, hiddenAt *time.Time
		if err := rows.Scan(
			&s.ID, &s.VersionID, &owner, &s.CreatedBy, &s.Name, &s.Query, &s.Filters, &s.IsGlobal,
			&s.CreatedAt, &disabledAt, &hiddenAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		s.OwnerID = owner
		s.DisabledAt = disabledAt
		s.HiddenAt = hiddenAt
		out = append(out, s)
	}
	return out, nil
}
