package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/service"
	"github.com/telhawk-systems/telhawk-stack/common/config"
)

// newTestAuthService creates a test auth service
func newTestAuthService(t *testing.T) *service.AuthService {
	cfg := config.GetConfig()
	cfg.Authenticate.JWTSecret = "test-jwt-secret-that-is-long-enough-for-hs256"
	cfg.Authenticate.JWTRefreshSecret = "test-refresh-secret-that-is-long-enough-for-hs256"
	cfg.Authenticate.AuditSecret = "test-audit-secret"

	repo := repository.NewInMemoryRepository()
	return service.NewAuthService(repo, nil)
}

// createTestUser creates a user and returns a valid token
func createTestUser(t *testing.T, svc *service.AuthService) (string, string) {
	t.Helper()

	req := &models.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Roles:    []string{"viewer"},
	}

	user, err := svc.CreateUser(context.Background(), req, "", "", "")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	loginReq := &models.LoginRequest{
		Username: user.Username,
		Password: "password123",
	}

	loginResp, err := svc.Login(context.Background(), loginReq, "", "")
	if err != nil {
		t.Fatalf("Failed to login test user: %v", err)
	}

	return user.ID, loginResp.AccessToken
}

func TestNewAuthMiddleware(t *testing.T) {
	svc := newTestAuthService(t)
	mw := NewAuthMiddleware(svc)

	if mw == nil {
		t.Fatal("NewAuthMiddleware returned nil")
	}
	if mw.authService != svc {
		t.Error("AuthMiddleware.authService not set correctly")
	}
}

func TestRequireAuth_Success(t *testing.T) {
	svc := newTestAuthService(t)
	mw := NewAuthMiddleware(svc)

	userID, token := createTestUser(t, svc)

	// Create test handler that checks context
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify user ID in context
		ctxUserID := r.Context().Value(UserIDKey)
		if ctxUserID != userID {
			t.Errorf("Expected user ID %s in context, got %v", userID, ctxUserID)
		}

		// Verify user object in context (for RBAC permission checking)
		ctxUser := UserFromContext(r.Context())
		if ctxUser == nil {
			t.Error("Expected user object in context, got nil")
		}

		w.WriteHeader(http.StatusOK)
	})

	// Create request with valid Bearer token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	// Execute middleware
	mw.RequireAuth(handler).ServeHTTP(rr, req)

	// Verify handler was called
	if !handlerCalled {
		t.Error("Handler was not called")
	}

	// Verify response
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
}

func TestRequireAuth_MissingAuthorizationHeader(t *testing.T) {
	svc := newTestAuthService(t)
	mw := NewAuthMiddleware(svc)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No Authorization header set

	rr := httptest.NewRecorder()
	mw.RequireAuth(handler).ServeHTTP(rr, req)

	if handlerCalled {
		t.Error("Handler should not be called when auth header is missing")
	}

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", status)
	}

	expectedBody := "Missing authorization header\n"
	if body := rr.Body.String(); body != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, body)
	}
}

func TestRequireAuth_InvalidAuthorizationHeaderFormat(t *testing.T) {
	svc := newTestAuthService(t)
	mw := NewAuthMiddleware(svc)

	tests := []struct {
		name   string
		header string
	}{
		{"NoBearer", "sometoken"},
		{"WrongScheme", "Basic dXNlcjpwYXNz"},
		{"ExtraSpaces", "Bearer  token"},
		{"OnlyBearer", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)

			rr := httptest.NewRecorder()
			mw.RequireAuth(handler).ServeHTTP(rr, req)

			if handlerCalled {
				t.Error("Handler should not be called with invalid auth header")
			}

			if status := rr.Code; status != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", status)
			}

			expectedBody := "Invalid authorization header\n"
			if body := rr.Body.String(); body != expectedBody {
				t.Errorf("Expected body %q, got %q", expectedBody, body)
			}
		})
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	svc := newTestAuthService(t)
	mw := NewAuthMiddleware(svc)

	tests := []struct {
		name  string
		token string
	}{
		{"Malformed", "not-a-jwt-token"},
		{"Empty", ""},
		{"Random", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)

			rr := httptest.NewRecorder()
			mw.RequireAuth(handler).ServeHTTP(rr, req)

			if handlerCalled {
				t.Error("Handler should not be called with invalid token")
			}

			if status := rr.Code; status != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", status)
			}

			expectedBody := "Invalid or expired token\n"
			if body := rr.Body.String(); body != expectedBody {
				t.Errorf("Expected body %q, got %q", expectedBody, body)
			}
		})
	}
}

// Note: RequireRole middleware was removed in favor of RequirePermission
// which uses the full RBAC system with UserRoles -> Role -> Permissions
