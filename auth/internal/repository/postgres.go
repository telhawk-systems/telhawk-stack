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
	pool, err := pgxpool.New(ctx, connString)
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

func (r *PostgresRepository) CreateUser(user *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (id, username, email, password_hash, roles, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.Roles, user.Enabled, user.CreatedAt, user.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "duplicate key value violates unique constraint" {
			return ErrUserExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetUserByUsername(username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, username, email, password_hash, roles, enabled, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Roles, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetUserByID(id string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, username, email, password_hash, roles, enabled, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Roles, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *PostgresRepository) UpdateUser(user *models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, roles = $5, enabled = $6
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Roles, user.Enabled,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *PostgresRepository) CreateSession(session *models.Session) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO sessions (id, user_id, access_token, refresh_token, expires_at, created_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.AccessToken, session.RefreshToken,
		session.ExpiresAt, session.CreatedAt, session.Revoked,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSession(refreshToken string) (*models.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, access_token, refresh_token, expires_at, created_at, revoked
		FROM sessions
		WHERE refresh_token = $1
	`

	var session models.Session
	err := r.pool.QueryRow(ctx, query, refreshToken).Scan(
		&session.ID, &session.UserID, &session.AccessToken, &session.RefreshToken,
		&session.ExpiresAt, &session.CreatedAt, &session.Revoked,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

func (r *PostgresRepository) RevokeSession(refreshToken string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

func (r *PostgresRepository) CreateHECToken(token *models.HECToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO hec_tokens (id, token, name, user_id, enabled, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var expiresAt *time.Time
	if !token.ExpiresAt.IsZero() {
		expiresAt = &token.ExpiresAt
	}

	_, err := r.pool.Exec(ctx, query,
		token.ID, token.Token, token.Name, token.UserID,
		token.Enabled, token.CreatedAt, expiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create HEC token: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetHECToken(token string) (*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, token, name, user_id, enabled, created_at, COALESCE(expires_at, '0001-01-01'::timestamp)
		FROM hec_tokens
		WHERE token = $1
	`

	var hecToken models.HECToken
	var expiresAt time.Time

	err := r.pool.QueryRow(ctx, query, token).Scan(
		&hecToken.ID, &hecToken.Token, &hecToken.Name, &hecToken.UserID,
		&hecToken.Enabled, &hecToken.CreatedAt, &expiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrHECTokenNotFound
		}
		return nil, fmt.Errorf("failed to get HEC token: %w", err)
	}

	if !expiresAt.IsZero() && expiresAt.Year() > 1 {
		hecToken.ExpiresAt = expiresAt
	}

	return &hecToken, nil
}

func (r *PostgresRepository) ListHECTokensByUser(userID string) ([]*models.HECToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, token, name, user_id, enabled, created_at, COALESCE(expires_at, '0001-01-01'::timestamp)
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
			&token.Enabled, &token.CreatedAt, &expiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan HEC token: %w", err)
		}

		if !expiresAt.IsZero() && expiresAt.Year() > 1 {
			token.ExpiresAt = expiresAt
		}

		tokens = append(tokens, &token)
	}

	return tokens, nil
}

func (r *PostgresRepository) RevokeHECToken(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE hec_tokens SET enabled = false WHERE token = $1`

	result, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to revoke HEC token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrHECTokenNotFound
	}

	return nil
}

func (r *PostgresRepository) LogAudit(entry *models.AuditLogEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

func (r *PostgresRepository) ListUsers() ([]*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, username, email, password_hash, roles, enabled, created_at, updated_at
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
			&user.Roles, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
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

func (r *PostgresRepository) DeleteUser(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
