package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/telhawk-systems/telhawk-stack/common/httputil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/service"
	"github.com/telhawk-systems/telhawk-stack/common/config"
)

// ============================================================================
// Test Setup
// ============================================================================

// Simple mock repository for handler tests
type testRepo struct {
	users     map[string]*models.User
	usersName map[string]*models.User
	sessions  map[string]*models.Session
	hecTokens map[string]*models.HECToken
	hecByID   map[string]*models.HECToken
}

func newTestRepo() *testRepo {
	return &testRepo{
		users:     make(map[string]*models.User),
		usersName: make(map[string]*models.User),
		sessions:  make(map[string]*models.Session),
		hecTokens: make(map[string]*models.HECToken),
		hecByID:   make(map[string]*models.HECToken)}
}

func (r *testRepo) CreateUser(ctx context.Context, user *models.User) error {
	if _, exists := r.usersName[user.Username]; exists {
		return repository.ErrUserExists
	}
	r.users[user.ID] = user
	r.usersName[user.Username] = user
	return nil
}

func (r *testRepo) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if user, ok := r.usersName[username]; ok {
		return user, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *testRepo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *testRepo) GetUserWithRoles(ctx context.Context, id string) (*models.User, error) {
	// For tests, just delegate to GetUserByID (no RBAC data loaded)
	return r.GetUserByID(ctx, id)
}

func (r *testRepo) GetUserPermissionsVersion(ctx context.Context, userID string) (int, error) {
	if user, ok := r.users[userID]; ok {
		return user.PermissionsVersion, nil
	}
	return 0, repository.ErrUserNotFound
}

func (r *testRepo) UpdateUser(ctx context.Context, user *models.User) error {
	if _, exists := r.users[user.ID]; !exists {
		return repository.ErrUserNotFound
	}
	r.users[user.ID] = user
	return nil
}

func (r *testRepo) ListUsers(ctx context.Context) ([]*models.User, error) {
	users := make([]*models.User, 0, len(r.users))
	for _, u := range r.users {
		users = append(users, u)
	}
	return users, nil
}

func (r *testRepo) ListUsersByScope(ctx context.Context, scopeType string, orgID, clientID *string) ([]*models.User, error) {
	// Simple mock - return all users
	return r.ListUsers(ctx)
}

func (r *testRepo) DeleteUser(ctx context.Context, id string) error {
	user, ok := r.users[id]
	if !ok {
		return repository.ErrUserNotFound
	}
	delete(r.usersName, user.Username)
	delete(r.users, id)
	return nil
}

func (r *testRepo) CreateSession(ctx context.Context, session *models.Session) error {
	r.sessions[session.RefreshToken] = session
	return nil
}

func (r *testRepo) GetSession(ctx context.Context, refreshToken string) (*models.Session, error) {
	if s, ok := r.sessions[refreshToken]; ok {
		return s, nil
	}
	return nil, repository.ErrSessionNotFound
}

func (r *testRepo) GetSessionByAccessToken(ctx context.Context, accessToken string) (*models.Session, error) {
	for _, session := range r.sessions {
		if session.AccessToken == accessToken {
			return session, nil
		}
	}
	return nil, repository.ErrSessionNotFound
}

func (r *testRepo) RevokeSession(ctx context.Context, refreshToken string) error {
	s, ok := r.sessions[refreshToken]
	if !ok {
		return repository.ErrSessionNotFound
	}
	now := time.Now()
	s.RevokedAt = &now
	return nil
}

func (r *testRepo) CreateHECToken(ctx context.Context, token *models.HECToken) error {
	r.hecTokens[token.Token] = token
	r.hecByID[token.ID] = token
	return nil
}

func (r *testRepo) GetHECToken(ctx context.Context, token string) (*models.HECToken, error) {
	if t, ok := r.hecTokens[token]; ok {
		return t, nil
	}
	return nil, repository.ErrHECTokenNotFound
}

func (r *testRepo) GetHECTokenByID(ctx context.Context, id string) (*models.HECToken, error) {
	if t, ok := r.hecByID[id]; ok {
		return t, nil
	}
	return nil, repository.ErrHECTokenNotFound
}

func (r *testRepo) ListHECTokensByUser(ctx context.Context, userID string) ([]*models.HECToken, error) {
	tokens := []*models.HECToken{}
	for _, t := range r.hecTokens {
		if t.UserID == userID {
			tokens = append(tokens, t)
		}
	}
	return tokens, nil
}

func (r *testRepo) ListAllHECTokens(ctx context.Context) ([]*models.HECToken, error) {
	tokens := make([]*models.HECToken, 0, len(r.hecTokens))
	for _, t := range r.hecTokens {
		tokens = append(tokens, t)
	}
	return tokens, nil
}

func (r *testRepo) RevokeHECToken(ctx context.Context, token string) error {
	t, ok := r.hecTokens[token]
	if !ok {
		return repository.ErrHECTokenNotFound
	}
	now := time.Now()
	t.RevokedAt = &now
	return nil
}

func (r *testRepo) LogAudit(ctx context.Context, entry *models.AuditLogEntry) error {
	return nil
}

// Organization operations
func (r *testRepo) GetOrganization(ctx context.Context, id string) (*models.Organization, error) {
	return nil, repository.ErrOrganizationNotFound
}

func (r *testRepo) ListOrganizations(ctx context.Context) ([]*models.Organization, error) {
	return []*models.Organization{}, nil
}

// Client operations
func (r *testRepo) GetClient(ctx context.Context, id string) (*models.Client, error) {
	return nil, repository.ErrClientNotFound
}

func (r *testRepo) ListClients(ctx context.Context) ([]*models.Client, error) {
	return []*models.Client{}, nil
}

func (r *testRepo) ListClientsByOrganization(ctx context.Context, orgID string) ([]*models.Client, error) {
	return []*models.Client{}, nil
}

func setupTestConfig() {
	cfg := config.GetConfig()
	cfg.Authenticate.JWTSecret = "test-secret-key-long-enough"
	cfg.Authenticate.JWTRefreshSecret = "test-refresh-secret-long"
	cfg.Authenticate.AuditSecret = "test-audit"
}

func setupHandler() *AuthHandler {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	return NewAuthHandler(svc)
}

// ============================================================================
// Health Check Tests
// ============================================================================

func TestHealthCheckHandler(t *testing.T) {
	handler := setupHandler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// ============================================================================
// Login Handler Tests
// ============================================================================

func TestLoginHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/login", nil)
		w := httptest.NewRecorder()

		handler.Login(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/login", strings.NewReader("invalid"))
		w := httptest.NewRecorder()

		handler.Login(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		handler := setupHandler()
		body, _ := json.Marshal(models.LoginRequest{
			Username: "nonexistent",
			Password: "pass"})
		req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Login(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})
}

// ============================================================================
// ValidateToken Handler Tests
// ============================================================================

func TestValidateTokenHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/validate", nil)
		w := httptest.NewRecorder()

		handler.ValidateToken(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/validate", strings.NewReader("bad"))
		w := httptest.NewRecorder()

		handler.ValidateToken(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// RefreshToken Handler Tests
// ============================================================================

func TestRefreshTokenHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/refresh", nil)
		w := httptest.NewRecorder()

		handler.RefreshToken(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/refresh", strings.NewReader("bad"))
		w := httptest.NewRecorder()

		handler.RefreshToken(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// RevokeToken Handler Tests
// ============================================================================

func TestRevokeTokenHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/revoke", nil)
		w := httptest.NewRecorder()

		handler.RevokeToken(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/revoke", strings.NewReader("bad"))
		w := httptest.NewRecorder()

		handler.RevokeToken(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// ValidateHECToken Handler Tests
// ============================================================================

func TestValidateHECTokenHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/validate-hec", nil)
		w := httptest.NewRecorder()

		handler.ValidateHECToken(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/validate-hec", strings.NewReader("bad"))
		w := httptest.NewRecorder()

		handler.ValidateHECToken(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// CreateUser Handler Tests
// ============================================================================

func TestCreateUserHandler(t *testing.T) {
	t.Run("unauthorized - no user ID", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/users/create", nil)
		w := httptest.NewRecorder()

		handler.CreateUser(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})

	t.Run("forbidden - not admin", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/users/create", nil)
		req.Header.Set("X-User-ID", "user-id")
		req.Header.Set("X-User-Roles", "viewer")
		w := httptest.NewRecorder()

		handler.CreateUser(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/users/create", strings.NewReader("bad"))
		req.Header.Set("X-User-ID", "admin")
		req.Header.Set("X-User-Roles", "admin")
		w := httptest.NewRecorder()

		handler.CreateUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// ListUsers Handler Tests
// ============================================================================

func TestListUsersHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/users", nil)
		w := httptest.NewRecorder()

		handler.ListUsers(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()

		handler.ListUsers(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// ============================================================================
// GetUser Handler Tests
// ============================================================================

func TestGetUserHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/users?id=123", nil)
		w := httptest.NewRecorder()

		handler.GetUser(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("missing user id", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()

		handler.GetUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/users?id=nonexistent", nil)
		w := httptest.NewRecorder()

		handler.GetUser(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

// ============================================================================
// UpdateUser Handler Tests
// ============================================================================

func TestUpdateUserHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/users?id=123", nil)
		w := httptest.NewRecorder()

		handler.UpdateUser(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("missing user id", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("PUT", "/users", nil)
		w := httptest.NewRecorder()

		handler.UpdateUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("PUT", "/users?id=123", strings.NewReader("bad"))
		req.Header.Set("X-User-ID", "admin")
		w := httptest.NewRecorder()

		handler.UpdateUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// DeleteUser Handler Tests
// ============================================================================

func TestDeleteUserHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/users?id=123", nil)
		w := httptest.NewRecorder()

		handler.DeleteUser(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("missing user id", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("DELETE", "/users", nil)
		w := httptest.NewRecorder()

		handler.DeleteUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// ResetPassword Handler Tests
// ============================================================================

func TestResetPasswordHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/reset-password?id=123", nil)
		w := httptest.NewRecorder()

		handler.ResetPassword(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("missing user id", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/reset-password", nil)
		w := httptest.NewRecorder()

		handler.ResetPassword(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/reset-password?id=123", strings.NewReader("bad"))
		req.Header.Set("X-User-ID", "admin")
		w := httptest.NewRecorder()

		handler.ResetPassword(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// CreateHECToken Handler Tests
// ============================================================================

func TestCreateHECTokenHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/hec/tokens", nil)
		w := httptest.NewRecorder()

		handler.CreateHECToken(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/hec/tokens", nil)
		w := httptest.NewRecorder()

		handler.CreateHECToken(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/hec/tokens", strings.NewReader("bad"))
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		handler.CreateHECToken(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// ListHECTokens Handler Tests
// ============================================================================

func TestListHECTokensHandler(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/hec/tokens", nil)
		w := httptest.NewRecorder()

		handler.ListHECTokens(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/hec/tokens", nil)
		w := httptest.NewRecorder()

		handler.ListHECTokens(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})
}

// ============================================================================
// RevokeHECToken Handler Tests
// ============================================================================

func TestRevokeHECTokenHandler2(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/hec/tokens/revoke", nil)
		w := httptest.NewRecorder()

		handler.RevokeHECTokenHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/hec/tokens/revoke", nil)
		w := httptest.NewRecorder()

		handler.RevokeHECTokenHandler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("POST", "/hec/tokens/revoke", strings.NewReader("bad"))
		req.Header.Set("X-User-ID", "user-123")
		w := httptest.NewRecorder()

		handler.RevokeHECTokenHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// ============================================================================
// RevokeHECTokenByID Handler Tests
// ============================================================================

func TestRevokeHECTokenByIDHandler2(t *testing.T) {
	t.Run("wrong method", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("GET", "/api/v1/hec/tokens/123/revoke", nil)
		w := httptest.NewRecorder()

		handler.RevokeHECTokenByIDHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405, got %d", w.Code)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		handler := setupHandler()
		req := httptest.NewRequest("DELETE", "/api/v1/hec/tokens/123/revoke", nil)
		w := httptest.NewRecorder()

		handler.RevokeHECTokenByIDHandler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})
}

// ============================================================================
// httputil.GetClientIP Helper Tests
// ============================================================================

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			remoteAddr: "5.6.7.8:1234",
			expected:   "1.2.3.4"},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "2.3.4.5"},
			remoteAddr: "5.6.7.8:1234",
			expected:   "2.3.4.5"},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "3.4.5.6:1234",
			expected:   "3.4.5.6:1234"},
		{
			name: "X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "1.1.1.1",
				"X-Real-IP":       "2.2.2.2"},
			remoteAddr: "3.3.3.3:1234",
			expected:   "1.1.1.1"}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := httputil.GetClientIP(req)
			if ip != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

// ============================================================================
// Additional ListHECTokens Tests (comprehensive coverage)
// ============================================================================

func TestListHECTokensHandler_AdminUser(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create test users
	user1 := &models.User{
		ID:       "user-1",
		Username: "user1",
		Email:    "user1@example.com",
		Roles:    []string{"viewer"}}
	user2 := &models.User{
		ID:       "user-2",
		Username: "user2",
		Email:    "user2@example.com",
		Roles:    []string{"viewer"}}

	// Add users to repo
	repo.CreateUser(context.Background(), user1)
	repo.CreateUser(context.Background(), user2)

	// Create tokens
	token1 := &models.HECToken{
		ID:     "token-1",
		Token:  "abc123456789xyz987654321",
		Name:   "Token 1",
		UserID: "user-1"}
	token2 := &models.HECToken{
		ID:     "token-2",
		Token:  "def456789012uvw654321098",
		Name:   "Token 2",
		UserID: "user-2"}
	repo.CreateHECToken(context.Background(), token1)
	repo.CreateHECToken(context.Background(), token2)

	// Admin request
	req := httptest.NewRequest("GET", "/api/v1/hec/tokens", nil)
	req.Header.Set("X-User-ID", "admin-user")
	req.Header.Set("X-User-Roles", "admin,viewer")
	w := httptest.NewRecorder()

	handler.ListHECTokens(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(data))
	}

	// Check that tokens are masked and include usernames for admin
	firstToken := data[0].(map[string]interface{})
	attrs := firstToken["attributes"].(map[string]interface{})
	if attrs["username"] == nil {
		t.Error("Admin should see usernames")
	}
	tokenValue := attrs["token"].(string)
	if !strings.Contains(tokenValue, "...") {
		t.Error("Token should be masked")
	}
}

func TestListHECTokensHandler_RegularUser(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create test user
	user1 := &models.User{
		ID:       "user-1",
		Username: "user1",
		Email:    "user1@example.com",
		Roles:    []string{"viewer"}}
	repo.CreateUser(context.Background(), user1)

	// Create tokens for this user and another user
	token1 := &models.HECToken{
		ID:     "token-1",
		Token:  "abc123456789xyz987654321",
		Name:   "My Token",
		UserID: "user-1"}
	token2 := &models.HECToken{
		ID:     "token-2",
		Token:  "def456789012uvw654321098",
		Name:   "Other User Token",
		UserID: "user-2"}
	repo.CreateHECToken(context.Background(), token1)
	repo.CreateHECToken(context.Background(), token2)

	// Regular user request
	req := httptest.NewRequest("GET", "/api/v1/hec/tokens", nil)
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-User-Roles", "viewer")
	w := httptest.NewRecorder()

	handler.ListHECTokens(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 1 {
		t.Errorf("Expected 1 token (only own tokens), got %d", len(data))
	}

	// Check that username is not included for regular users
	firstToken := data[0].(map[string]interface{})
	attrs := firstToken["attributes"].(map[string]interface{})
	if attrs["username"] != nil && attrs["username"] != "" {
		t.Error("Regular users should not see usernames")
	}
}

func TestListHECTokensHandler_EmptyResult(t *testing.T) {
	handler := setupHandler()

	req := httptest.NewRequest("GET", "/api/v1/hec/tokens", nil)
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("X-User-Roles", "viewer")
	w := httptest.NewRecorder()

	handler.ListHECTokens(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("Expected 0 tokens, got %d", len(data))
	}
}

// ============================================================================
// Additional RevokeHECTokenByIDHandler Tests
// ============================================================================

func TestRevokeHECTokenByIDHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create token
	token := &models.HECToken{
		ID:     "token-123",
		Token:  "abc123",
		Name:   "Test Token",
		UserID: "user-1"}
	repo.CreateHECToken(context.Background(), token)

	// User revoking their own token
	req := httptest.NewRequest("POST", "/api/v1/hec/tokens/token-123/revoke", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()

	handler.RevokeHECTokenByIDHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", w.Code)
	}

	// Verify token was revoked
	revokedToken, _ := repo.GetHECTokenByID(context.Background(), "token-123")
	if revokedToken.RevokedAt == nil {
		t.Error("Token should be revoked")
	}
}

func TestRevokeHECTokenByIDHandler_Unauthorized(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create token owned by user-1
	token := &models.HECToken{
		ID:     "token-123",
		Token:  "abc123",
		Name:   "Test Token",
		UserID: "user-1"}
	repo.CreateHECToken(context.Background(), token)

	// Different user trying to revoke
	req := httptest.NewRequest("POST", "/api/v1/hec/tokens/token-123/revoke", nil)
	req.Header.Set("X-User-ID", "user-2")
	req.Header.Set("X-User-Roles", "viewer")
	w := httptest.NewRecorder()

	handler.RevokeHECTokenByIDHandler(w, req)

	// Handler returns 400 for all service errors
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestRevokeHECTokenByIDHandler_MissingTokenID(t *testing.T) {
	handler := setupHandler()

	// Request without token ID in path
	req := httptest.NewRequest("POST", "/api/v1/hec/tokens//revoke", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()

	handler.RevokeHECTokenByIDHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestRevokeHECTokenByIDHandler_NotFound(t *testing.T) {
	handler := setupHandler()

	req := httptest.NewRequest("POST", "/api/v1/hec/tokens/nonexistent/revoke", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()

	handler.RevokeHECTokenByIDHandler(w, req)

	// Handler returns 400 for all service errors including not found
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// ============================================================================
// Additional ValidateHECToken Tests
// ============================================================================

func TestValidateHECTokenHandler_ValidToken(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create valid token
	token := &models.HECToken{
		ID:     "token-123",
		Token:  "valid-token-12345",
		Name:   "Test Token",
		UserID: "user-1"}
	repo.CreateHECToken(context.Background(), token)

	body, _ := json.Marshal(map[string]string{"token": "valid-token-12345"})
	req := httptest.NewRequest("POST", "/api/v1/auth/validate-hec", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ValidateHECToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response["valid"].(bool) {
		t.Error("Expected valid=true")
	}
}

func TestValidateHECTokenHandler_InvalidToken(t *testing.T) {
	handler := setupHandler()

	body, _ := json.Marshal(map[string]string{"token": "invalid-token"})
	req := httptest.NewRequest("POST", "/api/v1/auth/validate-hec", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ValidateHECToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["valid"].(bool) {
		t.Error("Expected valid=false for invalid token")
	}
}

func TestValidateHECTokenHandler_RevokedToken(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create and revoke token
	now := time.Now()
	token := &models.HECToken{
		ID:        "token-123",
		Token:     "revoked-token",
		Name:      "Test Token",
		UserID:    "user-1",
		RevokedAt: &now,
	}
	repo.CreateHECToken(context.Background(), token)

	body, _ := json.Marshal(map[string]string{"token": "revoked-token"})
	req := httptest.NewRequest("POST", "/api/v1/auth/validate-hec", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ValidateHECToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["valid"].(bool) {
		t.Error("Expected valid=false for revoked token")
	}
}

func TestValidateHECTokenHandler_MissingToken(t *testing.T) {
	handler := setupHandler()

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest("POST", "/api/v1/auth/validate-hec", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ValidateHECToken(w, req)

	// Handler returns 200 with valid:false for empty/invalid tokens
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["valid"].(bool) {
		t.Error("Expected valid=false for missing token")
	}
}

// ============================================================================
// Success Case Tests (Happy Path)
// ============================================================================

func TestLoginHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create a test user with hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &models.User{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Roles:        []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	// Test login
	body, _ := json.Marshal(models.LoginRequest{
		Username: "testuser",
		Password: "password123"})
	req := httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.LoginResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.AccessToken == "" {
		t.Error("Expected access token")
	}
	if response.RefreshToken == "" {
		t.Error("Expected refresh token")
	}
}

func TestCreateUserHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create request
	reqData := models.CreateUserRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "password123",
		Roles:    []string{"viewer"}}
	body, _ := json.Marshal(reqData)
	req := httptest.NewRequest("POST", "/api/v1/users/create", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "admin-user")
	req.Header.Set("X-User-Roles", "admin")
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})

	if attrs["username"] != "newuser" {
		t.Errorf("Expected username 'newuser', got %v", attrs["username"])
	}
	if attrs["email"] != "newuser@example.com" {
		t.Errorf("Expected email 'newuser@example.com', got %v", attrs["email"])
	}
}

func TestRefreshTokenHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create user and login to get refresh token
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &models.User{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Roles:        []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	loginResp, _ := svc.Login(context.Background(), &models.LoginRequest{
		Username: "testuser",
		Password: "password123"}, "127.0.0.1", "test-agent")

	// Test refresh token
	body, _ := json.Marshal(models.RefreshTokenRequest{
		RefreshToken: loginResp.RefreshToken})
	req := httptest.NewRequest("POST", "/refresh", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.RefreshToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.LoginResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.AccessToken == "" {
		t.Error("Expected new access token")
	}
}

func TestValidateTokenHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create user and login to get access token
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &models.User{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Roles:        []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	loginResp, _ := svc.Login(context.Background(), &models.LoginRequest{
		Username: "testuser",
		Password: "password123"}, "127.0.0.1", "test-agent")

	// Test validate token
	body, _ := json.Marshal(models.ValidateTokenRequest{
		Token: loginResp.AccessToken})
	req := httptest.NewRequest("POST", "/validate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ValidateToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.ValidateTokenResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.UserID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got %s", response.UserID)
	}
	if len(response.Roles) == 0 {
		t.Error("Expected roles to be populated")
	}
}

func TestRevokeTokenHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create user and login to get refresh token
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &models.User{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Roles:        []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	loginResp, _ := svc.Login(context.Background(), &models.LoginRequest{
		Username: "testuser",
		Password: "password123"}, "127.0.0.1", "test-agent")

	// Test revoke token
	body, _ := json.Marshal(models.RevokeTokenRequest{
		Token: loginResp.RefreshToken})
	req := httptest.NewRequest("POST", "/revoke", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.RevokeToken(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify token is revoked
	session, _ := repo.GetSession(context.Background(), loginResp.RefreshToken)
	if session.RevokedAt == nil {
		t.Error("Expected session to be revoked")
	}
}

func TestGetUserHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create test user
	user := &models.User{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	// Get user
	req := httptest.NewRequest("GET", "/users?id=user-123", nil)
	w := httptest.NewRecorder()

	handler.GetUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})

	if attrs["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got %v", attrs["username"])
	}
}

func TestUpdateUserHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create test user
	user := &models.User{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	// Update user
	updateReq := models.UpdateUserRequest{
		Email: "newemail@example.com",
		Roles: []string{"admin", "viewer"}}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/users?id=user-123", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "admin-user")
	w := httptest.NewRecorder()

	handler.UpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})

	if attrs["email"] != "newemail@example.com" {
		t.Errorf("Expected email 'newemail@example.com', got %v", attrs["email"])
	}
}

func TestDeleteUserHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create test user
	user := &models.User{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	// Delete user
	req := httptest.NewRequest("DELETE", "/users?id=user-123", nil)
	req.Header.Set("X-User-ID", "admin-user")
	w := httptest.NewRecorder()

	handler.DeleteUser(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify user is deleted
	_, err := repo.GetUserByID(context.Background(), "user-123")
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Error("Expected user to be deleted")
	}
}

func TestResetPasswordHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
	user := &models.User{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Roles:        []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	// Reset password
	resetReq := models.ResetPasswordRequest{
		NewPassword: "newpassword123"}
	body, _ := json.Marshal(resetReq)
	req := httptest.NewRequest("POST", "/reset-password?id=user-123", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "admin-user")
	w := httptest.NewRecorder()

	handler.ResetPassword(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify password was changed using bcrypt
	updatedUser, _ := repo.GetUserByID(context.Background(), "user-123")
	err := bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte("newpassword123"))
	if err != nil {
		t.Error("Expected password to be updated")
	}
}

func TestCreateHECTokenHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create test user
	user := &models.User{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	// Create HEC token
	createReq := models.CreateHECTokenRequest{
		Name:     "My Test Token",
		ClientID: "00000000-0000-0000-0000-000000000011"} // Default client
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/hec/tokens", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "user-123")
	w := httptest.NewRecorder()

	handler.CreateHECToken(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})

	if attrs["name"] != "My Test Token" {
		t.Errorf("Expected name 'My Test Token', got %v", attrs["name"])
	}
	if attrs["token"] == nil || attrs["token"] == "" {
		t.Error("Expected token to be returned")
	}
	if attrs["user_id"] != "user-123" {
		t.Errorf("Expected user_id 'user-123', got %v", attrs["user_id"])
	}
}

func TestRevokeHECTokenHandler_Success(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create HEC token
	token := &models.HECToken{
		ID:     "token-123",
		Token:  "test-token-value",
		Name:   "Test Token",
		UserID: "user-123"}
	repo.CreateHECToken(context.Background(), token)

	// Revoke token
	revokeReq := models.RevokeHECTokenRequest{
		Token: "test-token-value"}
	body, _ := json.Marshal(revokeReq)
	req := httptest.NewRequest("POST", "/hec/tokens/revoke", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "user-123")
	w := httptest.NewRecorder()

	handler.RevokeHECTokenHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify token is revoked
	revokedToken, _ := repo.GetHECToken(context.Background(), "test-token-value")
	if revokedToken.RevokedAt == nil {
		t.Error("Expected token to be revoked")
	}
}

func TestUpdateUserHandler_PatchMethod(t *testing.T) {
	setupTestConfig()
	repo := newTestRepo()
	svc := service.NewAuthService(repo, nil)
	handler := NewAuthHandler(svc)

	// Create test user
	user := &models.User{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"viewer"}}
	repo.CreateUser(context.Background(), user)

	// Update user with PATCH method
	updateReq := models.UpdateUserRequest{
		Email: "patched@example.com"}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PATCH", "/users?id=user-123", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "admin-user")
	w := httptest.NewRecorder()

	handler.UpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListUsersHandler_InternalError(t *testing.T) {
	// This test would require a mock that returns an error, which is difficult with our simple testRepo
	// Skipping for now, but in a real scenario we'd test internal server error handling
	t.Skip("Requires mock that can return errors from ListUsers")
}

func TestCreateHECTokenHandler_InternalError(t *testing.T) {
	// This test would require a mock that returns an error from CreateHECToken
	// Skipping for now
	t.Skip("Requires mock that can return errors from CreateHECToken")
}
