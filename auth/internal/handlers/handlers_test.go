package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/auth/internal/config"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/models"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/service"
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
		hecByID:   make(map[string]*models.HECToken),
	}
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

func setupHandler() *AuthHandler {
	repo := newTestRepo()
	cfg := &config.AuthConfig{
		JWTSecret:        "test-secret-key-long-enough",
		JWTRefreshSecret: "test-refresh-secret-long",
		AuditSecret:      "test-audit",
	}
	svc := service.NewAuthService(repo, nil, cfg)
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
			Password: "pass",
		})
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
// getClientIP Helper Tests
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
			expected:   "1.2.3.4",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "2.3.4.5"},
			remoteAddr: "5.6.7.8:1234",
			expected:   "2.3.4.5",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "3.4.5.6:1234",
			expected:   "3.4.5.6:1234",
		},
		{
			name: "X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "1.1.1.1",
				"X-Real-IP":       "2.2.2.2",
			},
			remoteAddr: "3.3.3.3:1234",
			expected:   "1.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, ip)
			}
		})
	}
}
