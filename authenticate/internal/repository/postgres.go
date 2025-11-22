package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
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

// =============================================================================
// USERS (versioned: id + version_id)
// =============================================================================

func (r *PostgresRepository) CreateUser(ctx context.Context, user *models.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (id, version_id, username, email, password_hash, roles, primary_tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.VersionID, user.Username, user.Email, user.PasswordHash,
		user.Roles, user.PrimaryTenantID,
	)

	if err != nil {
		// Check for unique constraint violation (23505)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get the latest version of the user by username
	query := `
		SELECT id, version_id, username, email, password_hash, roles, primary_tenant_id,
		       disabled_at, disabled_by, deleted_at, deleted_by
		FROM users
		WHERE username = $1 AND deleted_at IS NULL
		ORDER BY version_id DESC
		LIMIT 1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.VersionID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Roles, &user.PrimaryTenantID,
		&user.DisabledAt, &user.DisabledBy, &user.DeletedAt, &user.DeletedBy,
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

	// Get the latest version of the user by stable ID
	query := `
		SELECT id, version_id, username, email, password_hash, roles, primary_tenant_id,
		       disabled_at, disabled_by, deleted_at, deleted_by
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		ORDER BY version_id DESC
		LIMIT 1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.VersionID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Roles, &user.PrimaryTenantID,
		&user.DisabledAt, &user.DisabledBy, &user.DeletedAt, &user.DeletedBy,
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

	// For versioned tables, we update the existing row (for lifecycle changes)
	// or insert a new row with new version_id (for content changes)
	// This simple implementation updates in place - for true versioning,
	// caller should generate new version_id and insert
	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, roles = $5, primary_tenant_id = $6
		WHERE version_id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		user.VersionID, user.Username, user.Email, user.PasswordHash, user.Roles, user.PrimaryTenantID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *PostgresRepository) ListUsers(ctx context.Context) ([]*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get latest version of each user, ordered by id (created_at via UUIDv7)
	query := `
		SELECT DISTINCT ON (id) id, version_id, username, email, password_hash, roles, primary_tenant_id,
		       disabled_at, disabled_by, deleted_at, deleted_by
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY id DESC, version_id DESC
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
			&user.ID, &user.VersionID, &user.Username, &user.Email, &user.PasswordHash,
			&user.Roles, &user.PrimaryTenantID,
			&user.DisabledAt, &user.DisabledBy, &user.DeletedAt, &user.DeletedBy,
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

	// Soft delete - set deleted_at on all versions
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// =============================================================================
// SESSIONS (append-only: id only, no version_id)
// =============================================================================

func (r *PostgresRepository) CreateSession(ctx context.Context, session *models.Session) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO sessions (id, user_id, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.AccessToken, session.RefreshToken,
		session.ExpiresAt,
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
		SELECT id, user_id, access_token, refresh_token, expires_at, revoked_at, revoked_by
		FROM sessions
		WHERE refresh_token = $1
	`

	var session models.Session
	err := r.pool.QueryRow(ctx, query, refreshToken).Scan(
		&session.ID, &session.UserID, &session.AccessToken, &session.RefreshToken,
		&session.ExpiresAt, &session.RevokedAt, &session.RevokedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

func (r *PostgresRepository) GetSessionByAccessToken(ctx context.Context, accessToken string) (*models.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, access_token, refresh_token, expires_at, revoked_at, revoked_by
		FROM sessions
		WHERE access_token = $1
	`

	var session models.Session
	err := r.pool.QueryRow(ctx, query, accessToken).Scan(
		&session.ID, &session.UserID, &session.AccessToken, &session.RefreshToken,
		&session.ExpiresAt, &session.RevokedAt, &session.RevokedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by access token: %w", err)
	}

	return &session, nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `UPDATE sessions SET revoked_at = NOW() WHERE refresh_token = $1`

	result, err := r.pool.Exec(ctx, query, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// =============================================================================
// HEC TOKENS (append-only: id only, no version_id)
// =============================================================================

func (r *PostgresRepository) CreateHECToken(ctx context.Context, token *models.HECToken) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO hec_tokens (id, token, name, user_id, client_id, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		token.ID, token.Token, token.Name, token.UserID, token.ClientID, token.CreatedBy, token.ExpiresAt,
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
		SELECT id, token, name, user_id, client_id, created_by, expires_at,
		       disabled_at, disabled_by, revoked_at, revoked_by
		FROM hec_tokens
		WHERE token = $1
	`

	var hecToken models.HECToken
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&hecToken.ID, &hecToken.Token, &hecToken.Name, &hecToken.UserID,
		&hecToken.ClientID, &hecToken.CreatedBy, &hecToken.ExpiresAt,
		&hecToken.DisabledAt, &hecToken.DisabledBy,
		&hecToken.RevokedAt, &hecToken.RevokedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrHECTokenNotFound
		}
		return nil, fmt.Errorf("failed to get HEC token: %w", err)
	}

	return &hecToken, nil
}

// GetHECTokenByID retrieves an HEC token by its ID
func (r *PostgresRepository) GetHECTokenByID(ctx context.Context, id string) (*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, token, name, user_id, client_id, created_by, expires_at,
		       disabled_at, disabled_by, revoked_at, revoked_by
		FROM hec_tokens
		WHERE id = $1
	`

	var hecToken models.HECToken
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&hecToken.ID, &hecToken.Token, &hecToken.Name, &hecToken.UserID,
		&hecToken.ClientID, &hecToken.CreatedBy, &hecToken.ExpiresAt,
		&hecToken.DisabledAt, &hecToken.DisabledBy,
		&hecToken.RevokedAt, &hecToken.RevokedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrHECTokenNotFound
		}
		return nil, fmt.Errorf("failed to get HEC token: %w", err)
	}

	return &hecToken, nil
}

func (r *PostgresRepository) ListHECTokensByUser(ctx context.Context, userID string) ([]*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Order by id DESC (UUIDv7 = created_at)
	query := `
		SELECT id, token, name, user_id, client_id, created_by, expires_at,
		       disabled_at, disabled_by, revoked_at, revoked_by
		FROM hec_tokens
		WHERE user_id = $1
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list HEC tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.HECToken
	for rows.Next() {
		var token models.HECToken
		err := rows.Scan(
			&token.ID, &token.Token, &token.Name, &token.UserID,
			&token.ClientID, &token.CreatedBy, &token.ExpiresAt,
			&token.DisabledAt, &token.DisabledBy,
			&token.RevokedAt, &token.RevokedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan HEC token: %w", err)
		}
		tokens = append(tokens, &token)
	}

	return tokens, nil
}

func (r *PostgresRepository) ListAllHECTokens(ctx context.Context) ([]*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Order by id DESC (UUIDv7 = created_at)
	query := `
		SELECT id, token, name, user_id, client_id, created_by, expires_at,
		       disabled_at, disabled_by, revoked_at, revoked_by
		FROM hec_tokens
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all HEC tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*models.HECToken
	for rows.Next() {
		var token models.HECToken
		err := rows.Scan(
			&token.ID, &token.Token, &token.Name, &token.UserID,
			&token.ClientID, &token.CreatedBy, &token.ExpiresAt,
			&token.DisabledAt, &token.DisabledBy,
			&token.RevokedAt, &token.RevokedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan HEC token: %w", err)
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

// =============================================================================
// AUDIT LOG (append-only)
// =============================================================================

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
