package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/config"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// mockRepository implements repository.Repository for testing
type mockRepository struct {
	// User operations
	users           map[string]*models.User // ID -> User
	usersByUsername map[string]*models.User // Username -> User
	createUserErr   error
	getUserErr      error
	updateUserErr   error
	deleteUserErr   error

	// Session operations
	sessions         map[string]*models.Session // RefreshToken -> Session
	createSessionErr error
	getSessionErr    error
	revokeSessionErr error

	// HEC Token operations
	hecTokens         map[string]*models.HECToken // Token -> HECToken
	hecTokensByID     map[string]*models.HECToken // ID -> HECToken
	createHECTokenErr error
	getHECTokenErr    error
	revokeHECTokenErr error

	// Audit operations (for audit.Repository interface)
	logAuditEventErr error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:           make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
		sessions:        make(map[string]*models.Session),
		hecTokens:       make(map[string]*models.HECToken),
		hecTokensByID:   make(map[string]*models.HECToken)}
}

// User operations
func (m *mockRepository) CreateUser(ctx context.Context, user *models.User) error {
	if m.createUserErr != nil {
		return m.createUserErr
	}
	if _, exists := m.usersByUsername[user.Username]; exists {
		return repository.ErrUserExists
	}
	m.users[user.ID] = user
	m.usersByUsername[user.Username] = user
	return nil
}

func (m *mockRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	user, exists := m.usersByUsername[username]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	user, exists := m.users[id]
	if !exists {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) GetUserWithRoles(ctx context.Context, id string) (*models.User, error) {
	// For tests, just delegate to GetUserByID (no RBAC data loaded)
	return m.GetUserByID(ctx, id)
}

func (m *mockRepository) GetUserPermissionsVersion(ctx context.Context, userID string) (int, error) {
	user, exists := m.users[userID]
	if !exists {
		return 0, repository.ErrUserNotFound
	}
	return user.PermissionsVersion, nil
}

func (m *mockRepository) UpdateUser(ctx context.Context, user *models.User) error {
	if m.updateUserErr != nil {
		return m.updateUserErr
	}
	if _, exists := m.users[user.ID]; !exists {
		return repository.ErrUserNotFound
	}
	m.users[user.ID] = user
	m.usersByUsername[user.Username] = user
	return nil
}

func (m *mockRepository) ListUsers(ctx context.Context) ([]*models.User, error) {
	users := make([]*models.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *mockRepository) DeleteUser(ctx context.Context, id string) error {
	if m.deleteUserErr != nil {
		return m.deleteUserErr
	}
	user, exists := m.users[id]
	if !exists {
		return repository.ErrUserNotFound
	}
	delete(m.usersByUsername, user.Username)
	delete(m.users, id)
	return nil
}

// Session operations
func (m *mockRepository) CreateSession(ctx context.Context, session *models.Session) error {
	if m.createSessionErr != nil {
		return m.createSessionErr
	}
	m.sessions[session.RefreshToken] = session
	return nil
}

func (m *mockRepository) GetSession(ctx context.Context, refreshToken string) (*models.Session, error) {
	if m.getSessionErr != nil {
		return nil, m.getSessionErr
	}
	session, exists := m.sessions[refreshToken]
	if !exists {
		return nil, repository.ErrSessionNotFound
	}
	return session, nil
}

func (m *mockRepository) GetSessionByAccessToken(ctx context.Context, accessToken string) (*models.Session, error) {
	if m.getSessionErr != nil {
		return nil, m.getSessionErr
	}
	for _, session := range m.sessions {
		if session.AccessToken == accessToken {
			return session, nil
		}
	}
	return nil, repository.ErrSessionNotFound
}

func (m *mockRepository) RevokeSession(ctx context.Context, refreshToken string) error {
	if m.revokeSessionErr != nil {
		return m.revokeSessionErr
	}
	session, exists := m.sessions[refreshToken]
	if !exists {
		return repository.ErrSessionNotFound
	}
	now := time.Now()
	session.RevokedAt = &now
	return nil
}

// HEC Token operations
func (m *mockRepository) CreateHECToken(ctx context.Context, token *models.HECToken) error {
	if m.createHECTokenErr != nil {
		return m.createHECTokenErr
	}
	m.hecTokens[token.Token] = token
	m.hecTokensByID[token.ID] = token
	return nil
}

func (m *mockRepository) GetHECToken(ctx context.Context, token string) (*models.HECToken, error) {
	if m.getHECTokenErr != nil {
		return nil, m.getHECTokenErr
	}
	hecToken, exists := m.hecTokens[token]
	if !exists {
		return nil, repository.ErrHECTokenNotFound
	}
	return hecToken, nil
}

func (m *mockRepository) GetHECTokenByID(ctx context.Context, id string) (*models.HECToken, error) {
	if m.getHECTokenErr != nil {
		return nil, m.getHECTokenErr
	}
	hecToken, exists := m.hecTokensByID[id]
	if !exists {
		return nil, repository.ErrHECTokenNotFound
	}
	return hecToken, nil
}

func (m *mockRepository) ListHECTokensByUser(ctx context.Context, userID string) ([]*models.HECToken, error) {
	tokens := make([]*models.HECToken, 0)
	for _, token := range m.hecTokens {
		if token.UserID == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (m *mockRepository) ListAllHECTokens(ctx context.Context) ([]*models.HECToken, error) {
	tokens := make([]*models.HECToken, 0, len(m.hecTokens))
	for _, token := range m.hecTokens {
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (m *mockRepository) RevokeHECToken(ctx context.Context, token string) error {
	if m.revokeHECTokenErr != nil {
		return m.revokeHECTokenErr
	}
	hecToken, exists := m.hecTokens[token]
	if !exists {
		return repository.ErrHECTokenNotFound
	}
	now := time.Now()
	hecToken.RevokedAt = &now
	return nil
}

// Organization operations
func (m *mockRepository) GetOrganization(ctx context.Context, id string) (*models.Organization, error) {
	return nil, repository.ErrOrganizationNotFound
}

func (m *mockRepository) ListOrganizations(ctx context.Context) ([]*models.Organization, error) {
	return []*models.Organization{}, nil
}

// Client operations
func (m *mockRepository) GetClient(ctx context.Context, id string) (*models.Client, error) {
	return nil, repository.ErrClientNotFound
}

func (m *mockRepository) ListClients(ctx context.Context) ([]*models.Client, error) {
	return []*models.Client{}, nil
}

func (m *mockRepository) ListClientsByOrganization(ctx context.Context, orgID string) ([]*models.Client, error) {
	return []*models.Client{}, nil
}

// Audit operations (implements audit.Repository interface)
func (m *mockRepository) LogAudit(ctx context.Context, entry *models.AuditLogEntry) error {
	if m.logAuditEventErr != nil {
		return m.logAuditEventErr
	}
	return nil
}

// Helper to create test auth service
func setupTestService() (*AuthService, *mockRepository) {
	repo := newMockRepository()
	cfg := &config.AuthConfig{
		JWTSecret:        "test-jwt-secret-that-is-long-enough-for-hs256",
		JWTRefreshSecret: "test-refresh-secret-that-is-long-enough-for-hs256",
		AuditSecret:      "test-audit-secret"}
	service := NewAuthService(repo, nil, cfg)
	return service, repo
}

// ============================================================================
// User Management Tests
// ============================================================================

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.CreateUserRequest
		setupRepo     func(*mockRepository)
		expectError   bool
		validateUser  func(*testing.T, *models.User)
		errorContains string
	}{
		{
			name: "successful user creation",
			request: &models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Roles:    []string{"viewer"}},
			setupRepo:   func(m *mockRepository) {},
			expectError: false,
			validateUser: func(t *testing.T, user *models.User) {
				if user.Username != "testuser" {
					t.Errorf("Expected username testuser, got %s", user.Username)
				}
				if user.Email != "test@example.com" {
					t.Errorf("Expected email test@example.com, got %s", user.Email)
				}
				if len(user.Roles) != 1 || user.Roles[0] != "viewer" {
					t.Errorf("Expected roles [viewer], got %v", user.Roles)
				}
				// Verify password was hashed
				err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123"))
				if err != nil {
					t.Errorf("Password was not hashed correctly: %v", err)
				}
			}},
		{
			name: "default role assigned when no roles provided",
			request: &models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123"},
			setupRepo:   func(m *mockRepository) {},
			expectError: false,
			validateUser: func(t *testing.T, user *models.User) {
				if len(user.Roles) != 1 || user.Roles[0] != string(models.LegacyRoleViewer) {
					t.Errorf("Expected default role [viewer], got %v", user.Roles)
				}
			}},
		{
			name: "duplicate username",
			request: &models.CreateUserRequest{
				Username: "existing",
				Email:    "new@example.com",
				Password: "password123"},
			setupRepo: func(m *mockRepository) {
				// Pre-create a user with same username
				existingUser := &models.User{
					ID:       "existing-id",
					Username: "existing",
					Email:    "old@example.com"}
				m.users["existing-id"] = existingUser
				m.usersByUsername["existing"] = existingUser
			},
			expectError:   true,
			errorContains: "already exists"},
		{
			name: "repository error",
			request: &models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123"},
			setupRepo: func(m *mockRepository) {
				m.createUserErr = errors.New("database error")
			},
			expectError:   true,
			errorContains: "database error"}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			user, err := service.CreateUser(
				context.Background(),
				tt.request,
				"admin-id",
				"192.168.1.1",
				"test-agent",
			)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if user == nil {
				t.Fatal("Expected user but got nil")
			}

			if tt.validateUser != nil {
				tt.validateUser(t, user)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	// Create a valid password hash for testing
	validPassword := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.DefaultCost)

	tests := []struct {
		name         string
		request      *models.LoginRequest
		setupRepo    func(*mockRepository)
		expectError  bool
		validateResp func(*testing.T, *models.LoginResponse)
		errorIs      error
	}{
		{
			name: "successful login",
			request: &models.LoginRequest{
				Username: "testuser",
				Password: validPassword},
			setupRepo: func(m *mockRepository) {
				user := &models.User{
					ID:           "user-id",
					Username:     "testuser",
					Email:        "test@example.com",
					PasswordHash: string(hashedPassword),
					Roles:        []string{"viewer"}}
				m.users[user.ID] = user
				m.usersByUsername[user.Username] = user
			},
			expectError: false,
			validateResp: func(t *testing.T, resp *models.LoginResponse) {
				if resp.AccessToken == "" {
					t.Error("Expected access token")
				}
				if resp.RefreshToken == "" {
					t.Error("Expected refresh token")
				}
				if resp.TokenType != "Bearer" {
					t.Errorf("Expected TokenType Bearer, got %s", resp.TokenType)
				}
			}},
		{
			name: "user not found",
			request: &models.LoginRequest{
				Username: "nonexistent",
				Password: "password123"},
			setupRepo:   func(m *mockRepository) {},
			expectError: true,
			errorIs:     ErrInvalidCredentials},
		{
			name: "wrong password",
			request: &models.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword"},
			setupRepo: func(m *mockRepository) {
				user := &models.User{
					ID:           "user-id",
					Username:     "testuser",
					PasswordHash: string(hashedPassword),
					Roles:        []string{"viewer"}}
				m.users[user.ID] = user
				m.usersByUsername[user.Username] = user
			},
			expectError: true,
			errorIs:     ErrInvalidCredentials},
		{
			name: "disabled user",
			request: &models.LoginRequest{
				Username: "disabled",
				Password: validPassword},
			setupRepo: func(m *mockRepository) {
				now := time.Now()
				user := &models.User{
					ID:           "user-id",
					Username:     "disabled",
					PasswordHash: string(hashedPassword),
					Roles:        []string{"viewer"},
					DisabledAt:   &now}
				m.users[user.ID] = user
				m.usersByUsername[user.Username] = user
			},
			expectError: true,
			errorIs:     ErrInvalidCredentials},
		{
			name: "deleted user",
			request: &models.LoginRequest{
				Username: "deleted",
				Password: validPassword},
			setupRepo: func(m *mockRepository) {
				now := time.Now()
				user := &models.User{
					ID:           "user-id",
					Username:     "deleted",
					PasswordHash: string(hashedPassword),
					Roles:        []string{"viewer"},
					DeletedAt:    &now}
				m.users[user.ID] = user
				m.usersByUsername[user.Username] = user
			},
			expectError: true,
			errorIs:     ErrInvalidCredentials}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			resp, err := service.Login(
				context.Background(),
				tt.request,
				"192.168.1.1",
				"test-agent",
			)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if tt.errorIs != nil && !errors.Is(err, tt.errorIs) {
					t.Errorf("Expected error %v, got %v", tt.errorIs, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatal("Expected response but got nil")
			}

			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}

			// Verify session was created
			if len(repo.sessions) != 1 {
				t.Errorf("Expected 1 session, got %d", len(repo.sessions))
			}
		})
	}
}

func TestRefreshToken(t *testing.T) {
	tests := []struct {
		name         string
		refreshToken string
		setupRepo    func(*mockRepository)
		expectError  bool
		validateResp func(*testing.T, *models.LoginResponse)
		errorIs      error
	}{
		{
			name:         "successful token refresh",
			refreshToken: "valid-refresh-token",
			setupRepo: func(m *mockRepository) {
				user := &models.User{
					ID:       "user-id",
					Username: "testuser",
					Roles:    []string{"viewer"}}
				session := &models.Session{
					ID:           "session-id",
					UserID:       user.ID,
					RefreshToken: "valid-refresh-token",
					ExpiresAt:    time.Now().Add(24 * time.Hour)}
				m.users[user.ID] = user
				m.sessions[session.RefreshToken] = session
			},
			expectError: false,
			validateResp: func(t *testing.T, resp *models.LoginResponse) {
				if resp.AccessToken == "" {
					t.Error("Expected new access token")
				}
				if resp.RefreshToken != "valid-refresh-token" {
					t.Error("Expected same refresh token")
				}
			}},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid-token",
			setupRepo:    func(m *mockRepository) {},
			expectError:  true,
			errorIs:      ErrInvalidToken},
		{
			name:         "expired session",
			refreshToken: "expired-token",
			setupRepo: func(m *mockRepository) {
				session := &models.Session{
					ID:           "session-id",
					UserID:       "user-id",
					RefreshToken: "expired-token",
					ExpiresAt:    time.Now().Add(-24 * time.Hour), // Expired
				}
				m.sessions[session.RefreshToken] = session
			},
			expectError: true,
			errorIs:     ErrInvalidToken},
		{
			name:         "revoked session",
			refreshToken: "revoked-token",
			setupRepo: func(m *mockRepository) {
				now := time.Now()
				session := &models.Session{
					ID:           "session-id",
					UserID:       "user-id",
					RefreshToken: "revoked-token",
					ExpiresAt:    time.Now().Add(24 * time.Hour),
					RevokedAt:    &now}
				m.sessions[session.RefreshToken] = session
			},
			expectError: true,
			errorIs:     ErrInvalidToken}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			resp, err := service.RefreshToken(context.Background(), tt.refreshToken)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if tt.errorIs != nil && !errors.Is(err, tt.errorIs) {
					t.Errorf("Expected error %v, got %v", tt.errorIs, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatal("Expected response but got nil")
			}

			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	service, repo := setupTestService()
	ctx := context.Background()

	// Generate a valid token for testing
	validToken, err := service.tokenGen.GenerateAccessToken("user-123", []string{"admin"}, 1, "", "")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Create a session for the valid token (required for session validation)
	session := &models.Session{
		ID:           "session-123",
		UserID:       "user-123",
		AccessToken:  validToken,
		RefreshToken: "refresh-token-123",
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour)}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	tests := []struct {
		name         string
		token        string
		expectValid  bool
		expectUserID string
		expectRoles  []string
	}{
		{
			name:         "valid token with session",
			token:        validToken,
			expectValid:  true,
			expectUserID: "user-123",
			expectRoles:  []string{"admin"}},
		{
			name:        "invalid token",
			token:       "invalid.token.here",
			expectValid: false},
		{
			name:        "empty token",
			token:       "",
			expectValid: false}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.ValidateToken(ctx, tt.token)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if resp.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, resp.Valid)
			}

			if tt.expectValid {
				if resp.UserID != tt.expectUserID {
					t.Errorf("Expected UserID=%s, got %s", tt.expectUserID, resp.UserID)
				}
				if len(resp.Roles) != len(tt.expectRoles) {
					t.Errorf("Expected roles=%v, got %v", tt.expectRoles, resp.Roles)
				}
			}
		})
	}
}

func TestValidateToken_NoSession(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	// Generate a valid JWT token but don't create a session
	validToken, err := service.tokenGen.GenerateAccessToken("user-123", []string{"admin"}, 1, "", "")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	// Token should be invalid because no session exists
	resp, err := service.ValidateToken(ctx, validToken)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Valid {
		t.Error("Expected token to be invalid when no session exists")
	}
}

func TestValidateToken_RevokedSession(t *testing.T) {
	service, repo := setupTestService()
	ctx := context.Background()

	// Generate a valid token and create a revoked session
	validToken, err := service.tokenGen.GenerateAccessToken("user-123", []string{"admin"}, 1, "", "")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	now := time.Now()
	session := &models.Session{
		ID:           "session-123",
		UserID:       "user-123",
		AccessToken:  validToken,
		RefreshToken: "refresh-token-123",
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		RevokedAt:    &now, // Session is revoked
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Token should be invalid because session is revoked
	resp, err := service.ValidateToken(ctx, validToken)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Valid {
		t.Error("Expected token to be invalid when session is revoked")
	}
}

// ============================================================================
// HEC Token Tests
// ============================================================================

func TestValidateHECToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		setupRepo   func(*mockRepository)
		expectError bool
		expectToken bool
		errorIs     error
	}{
		{
			name:  "valid HEC token",
			token: "valid-hec-token",
			setupRepo: func(m *mockRepository) {
				hecToken := &models.HECToken{
					ID:     "token-id",
					UserID: "user-id",
					Name:   "Test Token",
					Token:  "valid-hec-token"}
				m.hecTokens[hecToken.Token] = hecToken
				m.hecTokensByID[hecToken.ID] = hecToken
			},
			expectError: false,
			expectToken: true},
		{
			name:        "token not found",
			token:       "nonexistent-token",
			setupRepo:   func(m *mockRepository) {},
			expectError: true},
		{
			name:  "revoked token",
			token: "revoked-token",
			setupRepo: func(m *mockRepository) {
				now := time.Now()
				hecToken := &models.HECToken{
					ID:        "token-id",
					UserID:    "user-id",
					Name:      "Revoked Token",
					Token:     "revoked-token",
					RevokedAt: &now}
				m.hecTokens[hecToken.Token] = hecToken
			},
			expectError: true,
			errorIs:     ErrInvalidToken}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			token, err := service.ValidateHECToken(
				context.Background(),
				tt.token,
				"192.168.1.1",
				"test-agent",
			)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if tt.errorIs != nil && !errors.Is(err, tt.errorIs) {
					t.Errorf("Expected error %v, got %v", tt.errorIs, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectToken && token == nil {
				t.Fatal("Expected token but got nil")
			}
		})
	}
}

func TestListUsers(t *testing.T) {
	service, repo := setupTestService()

	// Add some users
	users := []*models.User{
		{ID: "user-1", Username: "alice", Email: "alice@test.com"},
		{ID: "user-2", Username: "bob", Email: "bob@test.com"},
		{ID: "user-3", Username: "charlie", Email: "charlie@test.com"}}
	for _, user := range users {
		repo.users[user.ID] = user
		repo.usersByUsername[user.Username] = user
	}

	result, err := service.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 users, got %d", len(result))
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		setupRepo   func(*mockRepository)
		expectError bool
	}{
		{
			name:   "user found",
			userID: "user-123",
			setupRepo: func(m *mockRepository) {
				m.users["user-123"] = &models.User{
					ID:       "user-123",
					Username: "testuser",
					Email:    "test@example.com"}
			},
			expectError: false},
		{
			name:        "user not found",
			userID:      "nonexistent",
			setupRepo:   func(m *mockRepository) {},
			expectError: true}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			user, err := service.GetUser(context.Background(), tt.userID)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if user == nil {
				t.Fatal("Expected user but got nil")
			}

			if user.ID != tt.userID {
				t.Errorf("Expected user ID %s, got %s", tt.userID, user.ID)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		setupRepo   func(*mockRepository)
		expectError bool
	}{
		{
			name:   "successful deletion",
			userID: "user-123",
			setupRepo: func(m *mockRepository) {
				user := &models.User{
					ID:       "user-123",
					Username: "testuser",
					Email:    "test@example.com"}
				m.users[user.ID] = user
				m.usersByUsername[user.Username] = user
			},
			expectError: false},
		{
			name:        "user not found",
			userID:      "nonexistent",
			setupRepo:   func(m *mockRepository) {},
			expectError: true}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			err := service.DeleteUser(context.Background(), tt.userID, "admin-id", "192.168.1.1", "test-agent")

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify user was deleted
			if _, exists := repo.users[tt.userID]; exists {
				t.Error("Expected user to be deleted")
			}
		})
	}
}

func TestResetPassword(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		newPassword string
		setupRepo   func(*mockRepository)
		expectError bool
	}{
		{
			name:        "successful password reset",
			userID:      "user-123",
			newPassword: "newpassword456",
			setupRepo: func(m *mockRepository) {
				m.users["user-123"] = &models.User{
					ID:           "user-123",
					Username:     "testuser",
					PasswordHash: "old-hash"}
			},
			expectError: false},
		{
			name:        "user not found",
			userID:      "nonexistent",
			newPassword: "newpassword456",
			setupRepo:   func(m *mockRepository) {},
			expectError: true}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			err := service.ResetPassword(
				context.Background(),
				tt.userID,
				tt.newPassword,
				"admin-id",
				"192.168.1.1",
				"test-agent",
			)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify password was updated
			user := repo.users[tt.userID]
			err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.newPassword))
			if err != nil {
				t.Error("Password was not updated correctly")
			}
		})
	}
}

func TestCreateHECToken(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		tokenName     string
		setupRepo     func(*mockRepository)
		expectError   bool
		validateToken func(*testing.T, *models.HECToken)
	}{
		{
			name:        "successful token creation",
			userID:      "user-123",
			tokenName:   "Test Token",
			setupRepo:   func(m *mockRepository) {},
			expectError: false,
			validateToken: func(t *testing.T, token *models.HECToken) {
				if token.Name != "Test Token" {
					t.Errorf("Expected name 'Test Token', got %s", token.Name)
				}
				if token.UserID != "user-123" {
					t.Errorf("Expected userID user-123, got %s", token.UserID)
				}
				if token.Token == "" {
					t.Error("Expected token to be generated")
				}
			}},
		{
			name:        "token creation for any user",
			userID:      "any-user-id",
			tokenName:   "Another Token",
			setupRepo:   func(m *mockRepository) {},
			expectError: false}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			token, err := service.CreateHECToken(
				context.Background(),
				tt.userID,
				"00000000-0000-0000-0000-000000000011", // Default client
				tt.tokenName,
				"",
				"192.168.1.1",
				"test-agent",
			)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if token == nil {
				t.Fatal("Expected token but got nil")
			}

			if tt.validateToken != nil {
				tt.validateToken(t, token)
			}
		})
	}
}

func TestListHECTokensByUser(t *testing.T) {
	service, repo := setupTestService()

	// Create user
	repo.users["user-123"] = &models.User{ID: "user-123", Username: "testuser"}

	// Create tokens for user
	tokens := []*models.HECToken{
		{ID: "token-1", UserID: "user-123", Name: "Token 1", Token: "tok1"},
		{ID: "token-2", UserID: "user-123", Name: "Token 2", Token: "tok2"},
		{ID: "token-3", UserID: "other-user", Name: "Token 3", Token: "tok3"}}
	for _, token := range tokens {
		repo.hecTokens[token.Token] = token
		repo.hecTokensByID[token.ID] = token
	}

	result, err := service.ListHECTokensByUser(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 tokens for user-123, got %d", len(result))
	}
}

func TestRevokeToken(t *testing.T) {
	tests := []struct {
		name         string
		refreshToken string
		setupRepo    func(*mockRepository)
		expectError  bool
	}{
		{
			name:         "successful revocation",
			refreshToken: "valid-token",
			setupRepo: func(m *mockRepository) {
				m.sessions["valid-token"] = &models.Session{
					ID:           "session-id",
					RefreshToken: "valid-token",
					ExpiresAt:    time.Now().Add(24 * time.Hour)}
			},
			expectError: false},
		{
			name:         "token not found",
			refreshToken: "nonexistent",
			setupRepo:    func(m *mockRepository) {},
			expectError:  true}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			err := service.RevokeToken(context.Background(), tt.refreshToken)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify session was revoked
			session := repo.sessions[tt.refreshToken]
			if session.RevokedAt == nil {
				t.Error("Expected session to be revoked")
			}
		})
	}
}

func TestUpdateUserDetails(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		request      *models.UpdateUserRequest
		setupRepo    func(*mockRepository)
		expectError  bool
		validateUser func(*testing.T, *models.User)
	}{
		{
			name:   "update email",
			userID: "user-123",
			request: &models.UpdateUserRequest{
				Email: "newemail@example.com"},
			setupRepo: func(m *mockRepository) {
				m.users["user-123"] = &models.User{
					ID:       "user-123",
					Username: "testuser",
					Email:    "oldemail@example.com",
					Roles:    []string{"viewer"}}
			},
			expectError: false,
			validateUser: func(t *testing.T, user *models.User) {
				if user.Email != "newemail@example.com" {
					t.Errorf("Expected email newemail@example.com, got %s", user.Email)
				}
			}},
		{
			name:   "update roles",
			userID: "user-123",
			request: &models.UpdateUserRequest{
				Roles: []string{"admin", "editor"}},
			setupRepo: func(m *mockRepository) {
				m.users["user-123"] = &models.User{
					ID:       "user-123",
					Username: "testuser",
					Email:    "test@example.com",
					Roles:    []string{"viewer"}}
			},
			expectError: false,
			validateUser: func(t *testing.T, user *models.User) {
				if len(user.Roles) != 2 || user.Roles[0] != "admin" || user.Roles[1] != "editor" {
					t.Errorf("Expected roles [admin, editor], got %v", user.Roles)
				}
			}},
		{
			name:   "user not found",
			userID: "nonexistent",
			request: &models.UpdateUserRequest{
				Email: "test@example.com"},
			setupRepo:   func(m *mockRepository) {},
			expectError: true}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			user, err := service.UpdateUserDetails(
				context.Background(),
				tt.userID,
				tt.request,
				"admin-id",
				"192.168.1.1",
				"test-agent",
			)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if user == nil {
				t.Fatal("Expected user but got nil")
			}

			if tt.validateUser != nil {
				tt.validateUser(t, user)
			}
		})
	}
}

func TestListAllHECTokensWithUsernames(t *testing.T) {
	service, repo := setupTestService()

	// Create users
	users := []*models.User{
		{ID: "user-1", Username: "alice"},
		{ID: "user-2", Username: "bob"}}
	for _, user := range users {
		repo.users[user.ID] = user
		repo.usersByUsername[user.Username] = user
	}

	// Create tokens
	tokens := []*models.HECToken{
		{ID: "token-1", UserID: "user-1", Name: "Alice Token 1", Token: "tok1"},
		{ID: "token-2", UserID: "user-1", Name: "Alice Token 2", Token: "tok2"},
		{ID: "token-3", UserID: "user-2", Name: "Bob Token", Token: "tok3"}}
	for _, token := range tokens {
		repo.hecTokens[token.Token] = token
		repo.hecTokensByID[token.ID] = token
	}

	usernames, hecTokens, err := service.ListAllHECTokensWithUsernames(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(usernames) != 2 {
		t.Errorf("Expected 2 users, got %d", len(usernames))
	}

	if len(hecTokens) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(hecTokens))
	}

	if usernames["user-1"] != "alice" {
		t.Errorf("Expected username alice for user-1, got %s", usernames["user-1"])
	}
}

func TestRevokeHECTokenByUser(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		userID      string
		setupRepo   func(*mockRepository)
		expectError bool
	}{
		{
			name:   "successful revocation",
			token:  "test-token",
			userID: "user-123",
			setupRepo: func(m *mockRepository) {
				m.hecTokens["test-token"] = &models.HECToken{
					ID:     "token-id",
					UserID: "user-123",
					Token:  "test-token",
					Name:   "Test Token"}
				m.hecTokensByID["token-id"] = m.hecTokens["test-token"]
			},
			expectError: false},
		{
			name:        "token not found",
			token:       "nonexistent",
			userID:      "user-123",
			setupRepo:   func(m *mockRepository) {},
			expectError: true}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			err := service.RevokeHECTokenByUser(
				context.Background(),
				tt.token,
				tt.userID,
				"192.168.1.1",
				"test-agent",
			)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify token was revoked
			hecToken := repo.hecTokens[tt.token]
			if hecToken.RevokedAt == nil {
				t.Error("Expected token to be revoked")
			}
		})
	}
}

func TestRevokeHECTokenByID(t *testing.T) {
	tests := []struct {
		name        string
		tokenID     string
		userID      string
		setupRepo   func(*mockRepository)
		expectError bool
	}{
		{
			name:    "successful revocation",
			tokenID: "token-id",
			userID:  "user-123",
			setupRepo: func(m *mockRepository) {
				token := &models.HECToken{
					ID:     "token-id",
					UserID: "user-123",
					Token:  "test-token",
					Name:   "Test Token"}
				m.hecTokens["test-token"] = token
				m.hecTokensByID["token-id"] = token
			},
			expectError: false},
		{
			name:        "token not found",
			tokenID:     "nonexistent",
			userID:      "user-123",
			setupRepo:   func(m *mockRepository) {},
			expectError: true}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, repo := setupTestService()
			tt.setupRepo(repo)

			err := service.RevokeHECTokenByID(
				context.Background(),
				tt.tokenID,
				tt.userID,
				"192.168.1.1",
				"test-agent",
			)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify token was revoked
			hecToken := repo.hecTokensByID[tt.tokenID]
			if hecToken.RevokedAt == nil {
				t.Error("Expected token to be revoked")
			}
		})
	}
}

// ============================================================================
// Additional Error Case Tests
// ============================================================================

func TestResetPassword_UpdateUserError(t *testing.T) {
	service, repo := setupTestService()
	repo.users["user-123"] = &models.User{
		ID:           "user-123",
		Username:     "testuser",
		PasswordHash: "old-hash"}

	// Force UpdateUser to fail
	repo.updateUserErr = errors.New("database connection failed")

	err := service.ResetPassword(
		context.Background(),
		"user-123",
		"newpassword123",
		"admin-id",
		"192.168.1.1",
		"test-agent",
	)

	if err == nil {
		t.Fatal("Expected error when UpdateUser fails")
	}

	if err.Error() != "database connection failed" {
		t.Errorf("Expected 'database connection failed', got %v", err)
	}
}

func TestDeleteUser_GetUserError(t *testing.T) {
	service, repo := setupTestService()

	// Force GetUserByID to fail
	repo.getUserErr = errors.New("database error")

	err := service.DeleteUser(
		context.Background(),
		"user-123",
		"admin-id",
		"192.168.1.1",
		"test-agent",
	)

	if err == nil {
		t.Fatal("Expected error when GetUserByID fails")
	}
}

func TestDeleteUser_DeleteError(t *testing.T) {
	service, repo := setupTestService()
	repo.users["user-123"] = &models.User{
		ID:       "user-123",
		Username: "testuser"}

	// Force DeleteUser to fail
	repo.deleteUserErr = errors.New("cascade constraint violation")

	err := service.DeleteUser(
		context.Background(),
		"user-123",
		"admin-id",
		"192.168.1.1",
		"test-agent",
	)

	if err == nil {
		t.Fatal("Expected error when DeleteUser fails")
	}

	if err.Error() != "cascade constraint violation" {
		t.Errorf("Expected 'cascade constraint violation', got %v", err)
	}
}

func TestCreateHECToken_CreateError(t *testing.T) {
	service, repo := setupTestService()

	// Force CreateHECToken to fail
	repo.createHECTokenErr = errors.New("unique constraint violation")

	token, err := service.CreateHECToken(
		context.Background(),
		"user-123",
		"00000000-0000-0000-0000-000000000011", // Default client
		"Test Token",
		"never",
		"192.168.1.1",
		"test-agent",
	)

	if err == nil {
		t.Fatal("Expected error when CreateHECToken fails")
	}

	if token != nil {
		t.Error("Expected nil token when creation fails")
	}

	if err.Error() != "unique constraint violation" {
		t.Errorf("Expected 'unique constraint violation', got %v", err)
	}
}

func TestRevokeHECTokenByUser_RevokeError(t *testing.T) {
	service, repo := setupTestService()

	token := &models.HECToken{
		ID:     "token-id",
		UserID: "user-123",
		Token:  "test-token",
		Name:   "Test Token"}
	repo.hecTokens["test-token"] = token

	// Force RevokeHECToken to fail
	repo.revokeHECTokenErr = errors.New("database deadlock")

	err := service.RevokeHECTokenByUser(
		context.Background(),
		"test-token",
		"user-123",
		"192.168.1.1",
		"test-agent",
	)

	if err == nil {
		t.Fatal("Expected error when RevokeHECToken fails")
	}

	if err.Error() != "database deadlock" {
		t.Errorf("Expected 'database deadlock', got %v", err)
	}
}

func TestRevokeHECTokenByID_RevokeError(t *testing.T) {
	service, repo := setupTestService()

	token := &models.HECToken{
		ID:     "token-id",
		UserID: "user-123",
		Token:  "test-token",
		Name:   "Test Token"}
	repo.hecTokens["test-token"] = token
	repo.hecTokensByID["token-id"] = token

	// Force RevokeHECToken to fail
	repo.revokeHECTokenErr = errors.New("transaction timeout")

	err := service.RevokeHECTokenByID(
		context.Background(),
		"token-id",
		"user-123",
		"192.168.1.1",
		"test-agent",
	)

	if err == nil {
		t.Fatal("Expected error when RevokeHECToken fails")
	}

	if err.Error() != "transaction timeout" {
		t.Errorf("Expected 'transaction timeout', got %v", err)
	}
}

func TestRevokeHECTokenByUser_WrongOwner(t *testing.T) {
	service, repo := setupTestService()

	token := &models.HECToken{
		ID:     "token-id",
		UserID: "user-123",
		Token:  "test-token",
		Name:   "Test Token"}
	repo.hecTokens["test-token"] = token

	// Try to revoke token with wrong user
	err := service.RevokeHECTokenByUser(
		context.Background(),
		"test-token",
		"user-456", // Different user
		"192.168.1.1",
		"test-agent",
	)

	if err == nil {
		t.Fatal("Expected error when user doesn't own the token")
	}

	expectedErr := "unauthorized"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%v'", expectedErr, err)
	}
}

func TestRevokeHECTokenByID_WrongOwner(t *testing.T) {
	service, repo := setupTestService()

	token := &models.HECToken{
		ID:     "token-id",
		UserID: "user-123",
		Token:  "test-token",
		Name:   "Test Token"}
	repo.hecTokens["test-token"] = token
	repo.hecTokensByID["token-id"] = token

	// Try to revoke token with wrong user
	err := service.RevokeHECTokenByID(
		context.Background(),
		"token-id",
		"user-456", // Different user
		"192.168.1.1",
		"test-agent",
	)

	if err == nil {
		t.Fatal("Expected error when user doesn't own the token")
	}

	expectedErr := "unauthorized"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%v'", expectedErr, err)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
