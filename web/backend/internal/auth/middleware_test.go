package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createMockAuthServer creates a test HTTP server that simulates the auth service
// with customizable behavior for validate and refresh endpoints
func createMockAuthServer(
	validateFunc func(token string) (int, *ValidateResponse),
	refreshFunc func(refreshToken string) (int, *LoginResponse),
) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/validate":
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if validateFunc != nil {
				status, resp := validateFunc(req["token"])
				w.WriteHeader(status)
				if resp != nil {
					json.NewEncoder(w).Encode(resp)
				}
			} else {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ValidateResponse{
					Valid:  true,
					UserID: "test-user",
					Roles:  []string{"admin"},
				})
			}

		case "/api/v1/auth/refresh":
			var req RefreshRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if refreshFunc != nil {
				status, resp := refreshFunc(req.RefreshToken)
				w.WriteHeader(status)
				if resp != nil {
					json.NewEncoder(w).Encode(resp)
				}
			} else {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(LoginResponse{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
					ExpiresIn:    3600,
					TokenType:    "Bearer",
				})
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestMiddleware_Protect_ValidAccessToken(t *testing.T) {
	server := createMockAuthServer(
		func(token string) (int, *ValidateResponse) {
			if token != "valid-access-token" {
				return http.StatusOK, &ValidateResponse{Valid: false}
			}
			return http.StatusOK, &ValidateResponse{
				Valid:  true,
				UserID: "user-123",
				Roles:  []string{"admin", "user"},
			}
		},
		nil,
	)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "localhost", false)

	// Create a test handler that checks context values
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		if userID != "user-123" {
			t.Errorf("Expected userID 'user-123', got '%s'", userID)
		}

		roles := GetRoles(r.Context())
		if len(roles) != 2 || roles[0] != "admin" || roles[1] != "user" {
			t.Errorf("Expected roles ['admin', 'user'], got %v", roles)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := middleware.Protect(nextHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-access-token"})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "success" {
		t.Errorf("Expected body 'success', got '%s'", rr.Body.String())
	}
}

func TestMiddleware_Protect_MissingAccessToken(t *testing.T) {
	server := createMockAuthServer(nil, nil)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "localhost", false)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when access token is missing")
	})

	handler := middleware.Protect(nextHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	if rr.Body.String() != "Unauthorized\n" {
		t.Errorf("Expected body 'Unauthorized\\n', got '%s'", rr.Body.String())
	}
}

func TestMiddleware_Protect_InvalidAccessTokenWithValidRefresh(t *testing.T) {
	callCount := 0
	server := createMockAuthServer(
		func(token string) (int, *ValidateResponse) {
			callCount++
			// First call: invalid access token
			if callCount == 1 {
				return http.StatusOK, &ValidateResponse{Valid: false}
			}
			// Second call: valid new access token
			if token == "new-access-token" {
				return http.StatusOK, &ValidateResponse{
					Valid:  true,
					UserID: "user-456",
					Roles:  []string{"user"},
				}
			}
			return http.StatusOK, &ValidateResponse{Valid: false}
		},
		func(refreshToken string) (int, *LoginResponse) {
			if refreshToken == "valid-refresh-token" {
				return http.StatusOK, &LoginResponse{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
					ExpiresIn:    3600,
					TokenType:    "Bearer",
				}
			}
			return http.StatusUnauthorized, nil
		},
	)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "localhost", false)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		if userID != "user-456" {
			t.Errorf("Expected userID 'user-456', got '%s'", userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Protect(nextHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid-access-token"})
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh-token"})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Check that a new access token cookie was set
	cookies := rr.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "access_token" && cookie.Value == "new-access-token" {
			found = true
			if cookie.MaxAge != 3600 {
				t.Errorf("Expected MaxAge 3600, got %d", cookie.MaxAge)
			}
			if !cookie.HttpOnly {
				t.Error("Expected HttpOnly to be true")
			}
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("Expected SameSite Strict, got %v", cookie.SameSite)
			}
		}
	}
	if !found {
		t.Error("Expected new access_token cookie to be set")
	}
}

func TestMiddleware_Protect_InvalidAccessTokenWithMissingRefresh(t *testing.T) {
	server := createMockAuthServer(
		func(token string) (int, *ValidateResponse) {
			return http.StatusOK, &ValidateResponse{Valid: false}
		},
		nil,
	)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "localhost", false)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when refresh token is missing")
	})

	handler := middleware.Protect(nextHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid-access-token"})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestMiddleware_Protect_InvalidAccessTokenWithInvalidRefresh(t *testing.T) {
	server := createMockAuthServer(
		func(token string) (int, *ValidateResponse) {
			return http.StatusOK, &ValidateResponse{Valid: false}
		},
		func(refreshToken string) (int, *LoginResponse) {
			return http.StatusUnauthorized, nil
		},
	)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "localhost", false)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when refresh fails")
	})

	handler := middleware.Protect(nextHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid-access-token"})
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "invalid-refresh-token"})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestMiddleware_Protect_RefreshReturnsInvalidToken(t *testing.T) {
	server := createMockAuthServer(
		func(token string) (int, *ValidateResponse) {
			// All tokens are invalid
			return http.StatusOK, &ValidateResponse{Valid: false}
		},
		func(refreshToken string) (int, *LoginResponse) {
			// Refresh succeeds but returns an invalid token
			return http.StatusOK, &LoginResponse{
				AccessToken:  "invalid-new-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			}
		},
	)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "localhost", false)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when refreshed token is invalid")
	})

	handler := middleware.Protect(nextHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid-access-token"})
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh-token"})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestMiddleware_Protect_TokenValidationError(t *testing.T) {
	server := createMockAuthServer(
		func(token string) (int, *ValidateResponse) {
			return http.StatusInternalServerError, nil
		},
		nil,
	)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "localhost", false)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called when validation errors")
	})

	handler := middleware.Protect(nextHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "Valid UserID in context",
			ctx:      context.WithValue(context.Background(), UserIDKey, "user-123"),
			expected: "user-123",
		},
		{
			name:     "No UserID in context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "Invalid type in context",
			ctx:      context.WithValue(context.Background(), UserIDKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserID(tt.ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetRoles(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected []string
	}{
		{
			name:     "Valid Roles in context",
			ctx:      context.WithValue(context.Background(), RolesKey, []string{"admin", "user"}),
			expected: []string{"admin", "user"},
		},
		{
			name:     "No Roles in context",
			ctx:      context.Background(),
			expected: []string{},
		},
		{
			name:     "Invalid type in context",
			ctx:      context.WithValue(context.Background(), RolesKey, "admin"),
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRoles(tt.ctx)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d roles, got %d", len(tt.expected), len(result))
				return
			}
			for i, role := range result {
				if role != tt.expected[i] {
					t.Errorf("Expected role '%s' at index %d, got '%s'", tt.expected[i], i, role)
				}
			}
		})
	}
}

func TestMiddleware_setAccessTokenCookie(t *testing.T) {
	tests := []struct {
		name         string
		token        string
		expiresIn    int
		cookieDomain string
		cookieSecure bool
	}{
		{
			name:         "Standard cookie",
			token:        "test-token-123",
			expiresIn:    3600,
			cookieDomain: "example.com",
			cookieSecure: true,
		},
		{
			name:         "Insecure cookie",
			token:        "test-token-456",
			expiresIn:    7200,
			cookieDomain: "localhost",
			cookieSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockAuthServer(nil, nil)
			defer server.Close()

			authClient := NewClient(server.URL)
			middleware := NewMiddleware(authClient, tt.cookieDomain, tt.cookieSecure)

			rr := httptest.NewRecorder()
			middleware.setAccessTokenCookie(rr, tt.token, tt.expiresIn)

			cookies := rr.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("Expected 1 cookie, got %d", len(cookies))
			}

			cookie := cookies[0]
			if cookie.Name != "access_token" {
				t.Errorf("Expected cookie name 'access_token', got '%s'", cookie.Name)
			}
			if cookie.Value != tt.token {
				t.Errorf("Expected cookie value '%s', got '%s'", tt.token, cookie.Value)
			}
			if cookie.MaxAge != tt.expiresIn {
				t.Errorf("Expected MaxAge %d, got %d", tt.expiresIn, cookie.MaxAge)
			}
			if cookie.Path != "/" {
				t.Errorf("Expected Path '/', got '%s'", cookie.Path)
			}
			if cookie.Domain != tt.cookieDomain {
				t.Errorf("Expected Domain '%s', got '%s'", tt.cookieDomain, cookie.Domain)
			}
			if cookie.Secure != tt.cookieSecure {
				t.Errorf("Expected Secure %v, got %v", tt.cookieSecure, cookie.Secure)
			}
			if !cookie.HttpOnly {
				t.Error("Expected HttpOnly to be true")
			}
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("Expected SameSite Strict, got %v", cookie.SameSite)
			}
		})
	}
}

func TestMiddleware_setRefreshTokenCookie(t *testing.T) {
	server := createMockAuthServer(nil, nil)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "example.com", true)

	rr := httptest.NewRecorder()
	middleware.setRefreshTokenCookie(rr, "refresh-token-123")

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "refresh_token" {
		t.Errorf("Expected cookie name 'refresh_token', got '%s'", cookie.Name)
	}
	if cookie.Value != "refresh-token-123" {
		t.Errorf("Expected cookie value 'refresh-token-123', got '%s'", cookie.Value)
	}
	if cookie.MaxAge != 7*24*60*60 {
		t.Errorf("Expected MaxAge %d (7 days), got %d", 7*24*60*60, cookie.MaxAge)
	}
	if cookie.Path != "/" {
		t.Errorf("Expected Path '/', got '%s'", cookie.Path)
	}
	if cookie.Domain != "example.com" {
		t.Errorf("Expected Domain 'example.com', got '%s'", cookie.Domain)
	}
	if !cookie.Secure {
		t.Error("Expected Secure to be true")
	}
	if !cookie.HttpOnly {
		t.Error("Expected HttpOnly to be true")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("Expected SameSite Strict, got %v", cookie.SameSite)
	}
}

func TestMiddleware_clearAuthCookies(t *testing.T) {
	server := createMockAuthServer(nil, nil)
	defer server.Close()

	authClient := NewClient(server.URL)
	middleware := NewMiddleware(authClient, "example.com", true)

	rr := httptest.NewRecorder()
	middleware.clearAuthCookies(rr)

	cookies := rr.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies, got %d", len(cookies))
	}

	// Check access_token cookie
	var accessCookie, refreshCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			accessCookie = cookie
		} else if cookie.Name == "refresh_token" {
			refreshCookie = cookie
		}
	}

	if accessCookie == nil {
		t.Fatal("Expected access_token cookie to be set")
	}
	if accessCookie.Value != "" {
		t.Errorf("Expected empty access_token value, got '%s'", accessCookie.Value)
	}
	if accessCookie.MaxAge != -1 {
		t.Errorf("Expected MaxAge -1 for access_token, got %d", accessCookie.MaxAge)
	}

	if refreshCookie == nil {
		t.Fatal("Expected refresh_token cookie to be set")
	}
	if refreshCookie.Value != "" {
		t.Errorf("Expected empty refresh_token value, got '%s'", refreshCookie.Value)
	}
	if refreshCookie.MaxAge != -1 {
		t.Errorf("Expected MaxAge -1 for refresh_token, got %d", refreshCookie.MaxAge)
	}
}

// Integration-style test that verifies the entire middleware flow
func TestMiddleware_Protect_IntegrationFlow(t *testing.T) {
	// Create a mock auth server
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/validate":
			var req map[string]string
			json.NewDecoder(r.Body).Decode(&req)

			if req["token"] == "valid-token" || req["token"] == "new-valid-token" {
				json.NewEncoder(w).Encode(ValidateResponse{
					Valid:  true,
					UserID: "integration-user",
					Roles:  []string{"admin"},
				})
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ValidateResponse{Valid: false})
			}

		case "/api/v1/auth/refresh":
			var req RefreshRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.RefreshToken == "valid-refresh" {
				json.NewEncoder(w).Encode(LoginResponse{
					AccessToken:  "new-valid-token",
					RefreshToken: "new-refresh-token",
					ExpiresIn:    3600,
					TokenType:    "Bearer",
				})
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer authServer.Close()

	authClient := NewClient(authServer.URL)
	middleware := NewMiddleware(authClient, "localhost", false)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authenticated: " + userID))
	})

	handler := middleware.Protect(nextHandler)

	// Test 1: Valid token
	t.Run("Valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: "valid-token"})

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != "Authenticated: integration-user" {
			t.Errorf("Unexpected body: %s", rr.Body.String())
		}
	})

	// Test 2: Invalid token with valid refresh
	t.Run("Invalid token with valid refresh", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid-token"})
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh"})

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Verify new access token cookie was set
		cookies := rr.Result().Cookies()
		found := false
		for _, cookie := range cookies {
			if cookie.Name == "access_token" && cookie.Value == "new-valid-token" {
				found = true
			}
		}
		if !found {
			t.Error("Expected new access_token cookie to be set")
		}
	})
}
