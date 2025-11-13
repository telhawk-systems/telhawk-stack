package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/models"
)

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

func (r *PostgresRepository) CreateUser(ctx context.Context, user *models.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (id, username, email, password_hash, roles, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.Roles, user.CreatedAt,
	)

	if err != nil {
		if err.Error() == "duplicate key value violates unique constraint" {
			return ErrUserExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, username, email, password_hash, roles, created_at, disabled_at, disabled_by, deleted_at, deleted_by
		FROM users
		WHERE username = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Roles, &user.CreatedAt, &user.DisabledAt, &user.DisabledBy, &user.DeletedAt, &user.DeletedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, username, email, password_hash, roles, created_at, disabled_at, disabled_by, deleted_at, deleted_by
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Roles, &user.CreatedAt, &user.DisabledAt, &user.DisabledBy, &user.DeletedAt, &user.DeletedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *PostgresRepository) UpdateUser(ctx context.Context, user *models.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, roles = $5
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Roles,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session *models.Session) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO sessions (id, user_id, access_token, refresh_token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.AccessToken, session.RefreshToken,
		session.ExpiresAt, session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSession(ctx context.Context, refreshToken string) (*models.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, access_token, refresh_token, expires_at, created_at, revoked_at, revoked_by
		FROM sessions
		WHERE refresh_token = $1
	`

	var session models.Session
	err := r.pool.QueryRow(ctx, query, refreshToken).Scan(
		&session.ID, &session.UserID, &session.AccessToken, &session.RefreshToken,
		&session.ExpiresAt, &session.CreatedAt, &session.RevokedAt, &session.RevokedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `UPDATE sessions SET revoked = true WHERE refresh_token = $1`

	result, err := r.pool.Exec(ctx, query, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func (r *PostgresRepository) CreateHECToken(ctx context.Context, token *models.HECToken) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO hec_tokens (id, token, name, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		token.ID, token.Token, token.Name, token.UserID,
		token.CreatedAt, token.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create HEC token: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetHECToken(ctx context.Context, token string) (*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, token, name, user_id, created_at, COALESCE(expires_at, '0001-01-01'::timestamp),
		       disabled_at, disabled_by, revoked_at, revoked_by
		FROM hec_tokens
		WHERE token = $1
	`

	var hecToken models.HECToken
	var expiresAt time.Time

	err := r.pool.QueryRow(ctx, query, token).Scan(
		&hecToken.ID, &hecToken.Token, &hecToken.Name, &hecToken.UserID,
		&hecToken.CreatedAt, &expiresAt, &hecToken.DisabledAt, &hecToken.DisabledBy,
		&hecToken.RevokedAt, &hecToken.RevokedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrHECTokenNotFound
		}
		return nil, fmt.Errorf("failed to get HEC token: %w", err)
	}

	if !expiresAt.IsZero() && expiresAt.Year() > 1 {
		expiresCopy := expiresAt
		hecToken.ExpiresAt = &expiresCopy
	}

	return &hecToken, nil
}

// GetHECTokenByID retrieves an HEC token by its ID
func (r *PostgresRepository) GetHECTokenByID(ctx context.Context, id string) (*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, token, name, user_id, created_at, COALESCE(expires_at, '0001-01-01'::timestamp),
		       disabled_at, disabled_by, revoked_at, revoked_by
		FROM hec_tokens
		WHERE id = $1
	`

	var hecToken models.HECToken
	var expiresAt time.Time

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&hecToken.ID, &hecToken.Token, &hecToken.Name, &hecToken.UserID,
		&hecToken.CreatedAt, &expiresAt, &hecToken.DisabledAt, &hecToken.DisabledBy,
		&hecToken.RevokedAt, &hecToken.RevokedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrHECTokenNotFound
		}
		return nil, fmt.Errorf("failed to get HEC token: %w", err)
	}

	if !expiresAt.IsZero() && expiresAt.Year() > 1 {
		expiresCopy := expiresAt
		hecToken.ExpiresAt = &expiresCopy
	}

	return &hecToken, nil
}

func (r *PostgresRepository) ListHECTokensByUser(ctx context.Context, userID string) ([]*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, token, name, user_id, created_at, COALESCE(expires_at, '0001-01-01'::timestamp),
		       disabled_at, disabled_by, revoked_at, revoked_by
		FROM hec_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list HEC tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.HECToken
	for rows.Next() {
		var token models.HECToken
		var expiresAt time.Time

		err := rows.Scan(
			&token.ID, &token.Token, &token.Name, &token.UserID,
			&token.CreatedAt, &expiresAt, &token.DisabledAt, &token.DisabledBy,
			&token.RevokedAt, &token.RevokedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan HEC token: %w", err)
		}

		if !expiresAt.IsZero() && expiresAt.Year() > 1 {
			expiresCopy := expiresAt
			token.ExpiresAt = &expiresCopy
		}

		tokens = append(tokens, &token)
	}

	return tokens, nil
}

func (r *PostgresRepository) ListAllHECTokens(ctx context.Context) ([]*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, token, name, user_id, created_at, COALESCE(expires_at, '0001-01-01'::timestamp),
		       disabled_at, disabled_by, revoked_at, revoked_by
		FROM hec_tokens
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all HEC tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.HECToken
	for rows.Next() {
		var token models.HECToken
		var expiresAt time.Time

		err := rows.Scan(
			&token.ID, &token.Token, &token.Name, &token.UserID,
			&token.CreatedAt, &expiresAt, &token.DisabledAt, &token.DisabledBy,
			&token.RevokedAt, &token.RevokedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan HEC token: %w", err)
		}

		if !expiresAt.IsZero() && expiresAt.Year() > 1 {
			expiresCopy := expiresAt
			token.ExpiresAt = &expiresCopy
		}

		tokens = append(tokens, &token)
	}

	return tokens, nil
}

func (r *PostgresRepository) RevokeHECToken(ctx context.Context, token string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `UPDATE hec_tokens SET revoked_at = NOW() WHERE token = $1`

	result, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to revoke HEC token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrHECTokenNotFound
	}

	return nil
}

func (r *PostgresRepository) LogAudit(ctx context.Context, entry *models.AuditLogEntry) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var metadataJSON []byte
	var err error
	if entry.Metadata != nil {
		metadataJSON, err = json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO audit_log (
			timestamp, actor_type, actor_id, actor_name, action,
			resource_type, resource_id, ip_address, user_agent,
			result, error_message, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.pool.Exec(ctx, query,
		entry.Timestamp, entry.ActorType, entry.ActorID, entry.ActorName,
		entry.Action, entry.ResourceType, entry.ResourceID, entry.IPAddress,
		entry.UserAgent, entry.Result, entry.ErrorMessage, metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to log audit entry: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ListUsers(ctx context.Context) ([]*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, username, email, password_hash, roles, created_at,
		       disabled_at, disabled_by, deleted_at, deleted_by
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.Roles, &user.CreatedAt, &user.DisabledAt, &user.DisabledBy,
			&user.DeletedAt, &user.DeletedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

func (r *PostgresRepository) DeleteUser(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `DELETE FROM users WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}
