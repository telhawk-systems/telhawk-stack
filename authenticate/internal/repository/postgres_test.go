package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
)

// setupTestDatabase creates a PostgreSQL testcontainer and runs migrations
func setupTestDatabase(t *testing.T) (*PostgresRepository, func()) {
	ctx := context.Background()

	// Create PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("telhawk_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Run migrations
	if err := runMigrations(connStr); err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create repository
	repo, err := NewPostgresRepository(ctx, connStr)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		repo.pool.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return repo, cleanup
}

// runMigrations runs SQL migrations from the migrations directory
func runMigrations(connStr string) error {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Read migration file
	migrationPath := filepath.Join("..", "..", "migrations", "001_init.up.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration
	if _, err := db.Exec(string(migrationSQL)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// ============================================================================
// User Tests
// ============================================================================

func TestCreateUser(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()

	tests := []struct {
		name        string
		user        *models.User
		expectError bool
		errorType   error
	}{
		{
			name: "successful user creation",
			user: &models.User{
				ID: "11111111-1111-1111-1111-111111111111",

				VersionID:    "11111111-1111-1111-1111-111111111111",
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
				Roles:        []string{"viewer"}},
			expectError: false},
		{
			name: "duplicate username",
			user: &models.User{
				ID: "22222222-2222-2222-2222-222222222222",

				VersionID:    "22222222-2222-2222-2222-222222222222",
				Username:     "testuser", // Same as first
				Email:        "different@example.com",
				PasswordHash: "hashed_password",
				Roles:        []string{"viewer"}},
			expectError: true,
			errorType:   ErrUserExists},
		{
			name: "duplicate email",
			user: &models.User{
				ID: "33333333-3333-3333-3333-333333333333",

				VersionID:    "33333333-3333-3333-3333-333333333333",
				Username:     "differentuser",
				Email:        "test@example.com", // Same as first
				PasswordHash: "hashed_password",
				Roles:        []string{"viewer"}},
			expectError: true,
			errorType:   ErrUserExists}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := repo.CreateUser(ctx, tt.user)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify user was created
			retrieved, err := repo.GetUserByID(ctx, tt.user.ID)
			if err != nil {
				t.Fatalf("Failed to retrieve created user: %v", err)
			}

			if retrieved.Username != tt.user.Username {
				t.Errorf("Expected username %s, got %s", tt.user.Username, retrieved.Username)
			}
			if retrieved.Email != tt.user.Email {
				t.Errorf("Expected email %s, got %s", tt.user.Email, retrieved.Email)
			}
		})
	}
}

func TestGetUserByUsername(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create test user
	user := &models.User{
		ID: "44444444-4444-4444-4444-444444444444",

		VersionID:    "44444444-4444-4444-4444-444444444444",
		Username:     "gettest",
		Email:        "gettest@example.com",
		PasswordHash: "hashed_password",
		Roles:        []string{"viewer"}}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name        string
		username    string
		expectError bool
		errorType   error
	}{
		{
			name:        "user found",
			username:    "gettest",
			expectError: false},
		{
			name:        "user not found",
			username:    "nonexistent",
			expectError: true,
			errorType:   ErrUserNotFound}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := repo.GetUserByUsername(ctx, tt.username)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if retrieved.Username != user.Username {
				t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create test user
	user := &models.User{
		ID: "55555555-5555-5555-5555-555555555555",

		VersionID:    "55555555-5555-5555-5555-555555555555",
		Username:     "updatetest",
		Email:        "updatetest@example.com",
		PasswordHash: "hashed_password",
		Roles:        []string{"viewer"}}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Update user
	user.Email = "newemail@example.com"
	user.Roles = []string{"admin", "editor"}

	err := repo.UpdateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	if retrieved.Email != "newemail@example.com" {
		t.Errorf("Expected email newemail@example.com, got %s", retrieved.Email)
	}

	if len(retrieved.Roles) != 2 || retrieved.Roles[0] != "admin" {
		t.Errorf("Expected roles [admin, editor], got %v", retrieved.Roles)
	}
}

func TestDeleteUser(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create test user
	user := &models.User{
		ID: "66666666-6666-6666-6666-666666666666",

		VersionID:    "66666666-6666-6666-6666-666666666666",
		Username:     "deletetest",
		Email:        "deletetest@example.com",
		PasswordHash: "hashed_password",
		Roles:        []string{"viewer"}}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Delete user
	err := repo.DeleteUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify user is deleted (should not exist)
	_, err = repo.GetUserByID(ctx, user.ID)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound after deletion, got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create multiple users
	users := []*models.User{
		{
			ID: "77777777-7777-7777-7777-777777777777",

			VersionID:    "77777777-7777-7777-7777-777777777777",
			Username:     "listuser1",
			Email:        "listuser1@example.com",
			PasswordHash: "hash",
			Roles:        []string{"viewer"}},
		{
			ID: "88888888-8888-8888-8888-888888888888",

			VersionID:    "88888888-8888-8888-8888-888888888888",
			Username:     "listuser2",
			Email:        "listuser2@example.com",
			PasswordHash: "hash",
			Roles:        []string{"admin"}}}

	for _, user := range users {
		if err := repo.CreateUser(ctx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// List users (includes default admin from migration)
	retrieved, err := repo.ListUsers(ctx)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	// Should have at least our 2 test users + default admin
	if len(retrieved) < 3 {
		t.Errorf("Expected at least 3 users, got %d", len(retrieved))
	}
}

// ============================================================================
// Session Tests
// ============================================================================

func TestCreateSession(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create user first
	user := &models.User{
		ID: "99999999-9999-9999-9999-999999999999",

		VersionID:    "99999999-9999-9999-9999-999999999999",
		Username:     "sessionuser",
		Email:        "sessionuser@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create session
	session := &models.Session{
		ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",

		UserID:       user.ID,
		AccessToken:  "access_token_123",
		RefreshToken: "refresh_token_123",
		ExpiresAt:    time.Now().Add(24 * time.Hour)}

	err := repo.CreateSession(ctx, session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session was created
	retrieved, err := repo.GetSession(ctx, session.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if retrieved.UserID != user.ID {
		t.Errorf("Expected user_id %s, got %s", user.ID, retrieved.UserID)
	}
}

func TestGetSession(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create user and session
	user := &models.User{
		ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",

		VersionID:    "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Username:     "getsessionuser",
		Email:        "getsession@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user)

	session := &models.Session{
		ID: "cccccccc-cccc-cccc-cccc-cccccccccccc",

		UserID:       user.ID,
		AccessToken:  "access_token",
		RefreshToken: "refresh_token_get",
		ExpiresAt:    time.Now().Add(24 * time.Hour)}
	repo.CreateSession(ctx, session)

	tests := []struct {
		name         string
		refreshToken string
		expectError  bool
		errorType    error
	}{
		{
			name:         "session found",
			refreshToken: "refresh_token_get",
			expectError:  false},
		{
			name:         "session not found",
			refreshToken: "nonexistent_token",
			expectError:  true,
			errorType:    ErrSessionNotFound}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := repo.GetSession(ctx, tt.refreshToken)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if retrieved.RefreshToken != session.RefreshToken {
				t.Errorf("Expected refresh token %s, got %s", session.RefreshToken, retrieved.RefreshToken)
			}
		})
	}
}

func TestRevokeSession(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create user and session
	user := &models.User{
		ID: "dddddddd-dddd-dddd-dddd-dddddddddddd",

		VersionID:    "dddddddd-dddd-dddd-dddd-dddddddddddd",
		Username:     "revokesessionuser",
		Email:        "revokesession@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user)

	session := &models.Session{
		ID: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",

		UserID:       user.ID,
		AccessToken:  "access_token",
		RefreshToken: "refresh_token_revoke",
		ExpiresAt:    time.Now().Add(24 * time.Hour)}
	repo.CreateSession(ctx, session)

	// Revoke session
	err := repo.RevokeSession(ctx, session.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to revoke session: %v", err)
	}

	// Verify session is revoked
	retrieved, err := repo.GetSession(ctx, session.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to get revoked session: %v", err)
	}

	if retrieved.RevokedAt == nil {
		t.Error("Expected session to be revoked (RevokedAt should be set)")
	}

	if !retrieved.IsActive() {
		// This is correct - revoked session should NOT be active
	} else {
		t.Error("Expected revoked session to not be active")
	}
}

// ============================================================================
// HEC Token Tests
// ============================================================================

func TestCreateHECToken(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create user first
	user := &models.User{
		ID: "ffffffff-ffff-ffff-ffff-ffffffffffff",

		VersionID:    "ffffffff-ffff-ffff-ffff-ffffffffffff",
		Username:     "hectokenuser",
		Email:        "hectoken@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user)

	// Create HEC token
	token := &models.HECToken{
		ID:        "00000001-0001-0001-0001-000000000001",
		UserID:    user.ID,
		Token:     "test_hec_token_123",
		Name:      "Test Token",
		ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
		CreatedBy: user.ID}

	err := repo.CreateHECToken(ctx, token)
	if err != nil {
		t.Fatalf("Failed to create HEC token: %v", err)
	}

	// Verify token was created
	retrieved, err := repo.GetHECToken(ctx, token.Token)
	if err != nil {
		t.Fatalf("Failed to retrieve HEC token: %v", err)
	}

	if retrieved.Name != token.Name {
		t.Errorf("Expected name %s, got %s", token.Name, retrieved.Name)
	}
}

func TestGetHECToken(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create user and token
	user := &models.User{
		ID: "00000002-0002-0002-0002-000000000002",

		VersionID:    "00000002-0002-0002-0002-000000000002",
		Username:     "gethectokenuser",
		Email:        "gethectoken@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user)

	token := &models.HECToken{
		ID:        "00000003-0003-0003-0003-000000000003",
		UserID:    user.ID,
		Token:     "get_hec_token_test",
		Name:      "Get Test Token",
		ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
		CreatedBy: user.ID}
	repo.CreateHECToken(ctx, token)

	tests := []struct {
		name        string
		tokenValue  string
		expectError bool
		errorType   error
	}{
		{
			name:        "token found",
			tokenValue:  "get_hec_token_test",
			expectError: false},
		{
			name:        "token not found",
			tokenValue:  "nonexistent_token",
			expectError: true,
			errorType:   ErrHECTokenNotFound}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := repo.GetHECToken(ctx, tt.tokenValue)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if retrieved.Token != token.Token {
				t.Errorf("Expected token %s, got %s", token.Token, retrieved.Token)
			}
		})
	}
}

func TestListHECTokensByUser(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create users
	user1 := &models.User{
		ID: "00000004-0004-0004-0004-000000000004",

		VersionID:    "00000004-0004-0004-0004-000000000004",
		Username:     "listhecuser1",
		Email:        "listhec1@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	user2 := &models.User{
		ID: "00000005-0005-0005-0005-000000000005",

		VersionID:    "00000005-0005-0005-0005-000000000005",
		Username:     "listhecuser2",
		Email:        "listhec2@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user1)
	repo.CreateUser(ctx, user2)

	// Create tokens for user1
	tokens := []*models.HECToken{
		{
			ID:        "00000006-0006-0006-0006-000000000006",
			UserID:    user1.ID,
			Token:     "user1_token1",
			Name:      "User 1 Token 1",
			ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
			CreatedBy: user1.ID},
		{
			ID:        "00000007-0007-0007-0007-000000000007",
			UserID:    user1.ID,
			Token:     "user1_token2",
			Name:      "User 1 Token 2",
			ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
			CreatedBy: user1.ID},
		{
			ID:        "00000008-0008-0008-0008-000000000008",
			UserID:    user2.ID,
			Token:     "user2_token1",
			Name:      "User 2 Token 1",
			ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
			CreatedBy: user2.ID}}

	for _, token := range tokens {
		repo.CreateHECToken(ctx, token)
	}

	// List tokens for user1
	retrieved, err := repo.ListHECTokensByUser(ctx, user1.ID)
	if err != nil {
		t.Fatalf("Failed to list HEC tokens: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Expected 2 tokens for user1, got %d", len(retrieved))
	}
}

func TestRevokeHECToken(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create user and token
	user := &models.User{
		ID: "00000009-0009-0009-0009-000000000009",

		VersionID:    "00000009-0009-0009-0009-000000000009",
		Username:     "revokehectokenuser",
		Email:        "revokehectoken@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user)

	token := &models.HECToken{
		ID:        "0000000a-000a-000a-000a-00000000000a",
		UserID:    user.ID,
		Token:     "revoke_hec_token_test",
		Name:      "Revoke Test Token",
		ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
		CreatedBy: user.ID}
	repo.CreateHECToken(ctx, token)

	// Revoke token
	err := repo.RevokeHECToken(ctx, token.Token)
	if err != nil {
		t.Fatalf("Failed to revoke HEC token: %v", err)
	}

	// Verify token is revoked
	retrieved, err := repo.GetHECToken(ctx, token.Token)
	if err != nil {
		t.Fatalf("Failed to get revoked token: %v", err)
	}

	if retrieved.RevokedAt == nil {
		t.Error("Expected token to be revoked (RevokedAt should be set)")
	}

	if !retrieved.IsActive() {
		// This is correct - revoked token should NOT be active
	} else {
		t.Error("Expected revoked token to not be active")
	}
}

func TestGetHECTokenByID(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create user and token
	user := &models.User{
		ID: "0000000b-000b-000b-000b-00000000000b",

		VersionID:    "0000000b-000b-000b-000b-00000000000b",
		Username:     "gethectokenbyiduser",
		Email:        "gethectokenbyid@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user)

	token := &models.HECToken{
		ID:        "0000000c-000c-000c-000c-00000000000c",
		UserID:    user.ID,
		Token:     "get_by_id_token",
		Name:      "Get By ID Test Token",
		ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
		CreatedBy: user.ID}
	repo.CreateHECToken(ctx, token)

	// Test found
	t.Run("token_found", func(t *testing.T) {
		retrieved, err := repo.GetHECTokenByID(ctx, token.ID)
		if err != nil {
			t.Fatalf("Failed to get HEC token by ID: %v", err)
		}
		if retrieved.ID != token.ID {
			t.Errorf("Expected ID %s, got %s", token.ID, retrieved.ID)
		}
		if retrieved.Token != token.Token {
			t.Errorf("Expected token %s, got %s", token.Token, retrieved.Token)
		}
	})

	// Test not found
	t.Run("token_not_found", func(t *testing.T) {
		_, err := repo.GetHECTokenByID(ctx, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, ErrHECTokenNotFound) {
			t.Errorf("Expected ErrHECTokenNotFound, got %v", err)
		}
	})
}

func TestListAllHECTokens(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create two users and tokens
	user1 := &models.User{
		ID: "0000000d-000d-000d-000d-00000000000d",

		VersionID:    "0000000d-000d-000d-000d-00000000000d",
		Username:     "listalluser1",
		Email:        "listall1@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user1)

	user2 := &models.User{
		ID: "0000000e-000e-000e-000e-00000000000e",

		VersionID:    "0000000e-000e-000e-000e-00000000000e",
		Username:     "listalluser2",
		Email:        "listall2@example.com",
		PasswordHash: "hash",
		Roles:        []string{"viewer"}}
	repo.CreateUser(ctx, user2)

	// Create tokens for both users
	token1 := &models.HECToken{
		ID:        "0000000f-000f-000f-000f-00000000000f",
		UserID:    user1.ID,
		Token:     "list_all_token1",
		Name:      "List All Token 1",
		ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
		CreatedBy: user1.ID}
	repo.CreateHECToken(ctx, token1)

	token2 := &models.HECToken{
		ID:        "00000010-0010-0010-0010-000000000010",
		UserID:    user2.ID,
		Token:     "list_all_token2",
		Name:      "List All Token 2",
		ClientID:  "00000000-0000-0000-0000-000000000011", // Default Client
		CreatedBy: user2.ID}
	repo.CreateHECToken(ctx, token2)

	// List all tokens
	retrieved, err := repo.ListAllHECTokens(ctx)
	if err != nil {
		t.Fatalf("Failed to list all HEC tokens: %v", err)
	}

	// Should have at least 2 tokens (could have more from other tests if run in parallel)
	if len(retrieved) < 2 {
		t.Errorf("Expected at least 2 tokens, got %d", len(retrieved))
	}

	// Verify tokens have correct user IDs
	foundUser1 := false
	foundUser2 := false
	for _, token := range retrieved {
		if token.UserID == user1.ID {
			foundUser1 = true
		}
		if token.UserID == user2.ID {
			foundUser2 = true
		}
	}

	if !foundUser1 || !foundUser2 {
		t.Error("Expected to find tokens for both users")
	}
}

func TestLogAudit(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	defer cleanup()
	ctx := context.Background()

	// Create audit log entry
	entry := &models.AuditLogEntry{
		Timestamp:    time.Now(),
		ActorType:    "user",
		ActorID:      "test-user-id",
		ActorName:    "testuser",
		Action:       "login",
		ResourceType: "session",
		ResourceID:   "test-session-id",
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		Result:       "success",
		Metadata:     map[string]interface{}{"test": "data"}}

	err := repo.LogAudit(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to log audit entry: %v", err)
	}

	// Verify entry was written (we don't have a read method, but at least verify no error)
	// In production, you'd query audit_log table directly to verify
}

func TestClose(t *testing.T) {
	repo, cleanup := setupTestDatabase(t)
	// Don't use defer cleanup here since we're testing Close

	// Close should not panic
	repo.Close()

	// Cleanup container
	cleanup()
}
