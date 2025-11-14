package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/auth"
)

// createMockAuthClient creates a mock auth client with customizable behavior
func createMockAuthClient() (*httptest.Server, *auth.Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			var req map[string]string
			json.NewDecoder(r.Body).Decode(&req)

			if req["username"] == "admin" && req["password"] == "admin123" {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(auth.LoginResponse{
					AccessToken:  "valid-access-token",
					RefreshToken: "valid-refresh-token",
					ExpiresIn:    3600,
					TokenType:    "Bearer",
				})
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			}

		case "/api/v1/auth/revoke":
			var req map[string]string
			json.NewDecoder(r.Body).Decode(&req)

			if req["refresh_token"] == "valid-refresh-token" {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]bool{"success": true})
			} else {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid token"})
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server, auth.NewClient(server.URL)
}

func TestAuthHandler_GetCSRFToken(t *testing.T) {
	server, authClient := createMockAuthClient()
	defer server.Close()

	handler := NewAuthHandler(authClient, "localhost", false)

	req := httptest.NewRequest("GET", "/api/v1/auth/csrf", nil)
	rr := httptest.NewRecorder()

	handler.GetCSRFToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if csrfToken, ok := resp["csrf_token"].(string); !ok || csrfToken != "" {
		t.Errorf("Expected empty csrf_token, got %v", resp["csrf_token"])
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	server, authClient := createMockAuthClient()
	defer server.Close()

	handler := NewAuthHandler(authClient, "localhost", false)

	loginReq := LoginRequest{
		Username: "admin",
		Password: "admin123",
	}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if success, ok := resp["success"].(bool); !ok || !success {
		t.Error("Expected success to be true")
	}

	// Verify cookies were set
	cookies := rr.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies, got %d", len(cookies))
	}

	var accessToken, refreshToken *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			accessToken = cookie
		} else if cookie.Name == "refresh_token" {
			refreshToken = cookie
		}
	}

	if accessToken == nil {
		t.Error("Expected access_token cookie to be set")
	} else {
		if accessToken.Value != "valid-access-token" {
			t.Errorf("Expected access_token 'valid-access-token', got '%s'", accessToken.Value)
		}
		if accessToken.MaxAge != 3600 {
			t.Errorf("Expected MaxAge 3600, got %d", accessToken.MaxAge)
		}
		if !accessToken.HttpOnly {
			t.Error("Expected HttpOnly to be true")
		}
		if accessToken.SameSite != http.SameSiteStrictMode {
			t.Errorf("Expected SameSite Strict, got %v", accessToken.SameSite)
		}
	}

	if refreshToken == nil {
		t.Error("Expected refresh_token cookie to be set")
	} else {
		if refreshToken.Value != "valid-refresh-token" {
			t.Errorf("Expected refresh_token 'valid-refresh-token', got '%s'", refreshToken.Value)
		}
		expectedMaxAge := 7 * 24 * 60 * 60 // 7 days
		if refreshToken.MaxAge != expectedMaxAge {
			t.Errorf("Expected MaxAge %d, got %d", expectedMaxAge, refreshToken.MaxAge)
		}
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	server, authClient := createMockAuthClient()
	defer server.Close()

	handler := NewAuthHandler(authClient, "localhost", false)

	loginReq := LoginRequest{
		Username: "admin",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	// Verify no cookies were set
	cookies := rr.Result().Cookies()
	if len(cookies) != 0 {
		t.Errorf("Expected no cookies to be set, got %d", len(cookies))
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	server, authClient := createMockAuthClient()
	defer server.Close()

	handler := NewAuthHandler(authClient, "localhost", false)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Login(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	server, authClient := createMockAuthClient()
	defer server.Close()

	handler := NewAuthHandler(authClient, "localhost", false)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh-token"})
	rr := httptest.NewRecorder()

	handler.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if success, ok := resp["success"].(bool); !ok || !success {
		t.Error("Expected success to be true")
	}

	// Verify cookies were cleared
	cookies := rr.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies to be cleared, got %d", len(cookies))
	}

	for _, cookie := range cookies {
		if cookie.MaxAge != -1 {
			t.Errorf("Expected MaxAge -1 for %s, got %d", cookie.Name, cookie.MaxAge)
		}
		if cookie.Value != "" {
			t.Errorf("Expected empty value for %s, got '%s'", cookie.Name, cookie.Value)
		}
	}
}

func TestAuthHandler_Logout_NoRefreshToken(t *testing.T) {
	server, authClient := createMockAuthClient()
	defer server.Close()

	handler := NewAuthHandler(authClient, "localhost", false)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()

	handler.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Should still clear cookies even without refresh token
	cookies := rr.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies to be cleared, got %d", len(cookies))
	}
}

func TestAuthHandler_Me(t *testing.T) {
	server, authClient := createMockAuthClient()
	defer server.Close()

	handler := NewAuthHandler(authClient, "localhost", false)

	// Create request with user context
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	ctx := context.WithValue(req.Context(), auth.UserIDKey, "user-123")
	ctx = context.WithValue(ctx, auth.RolesKey, []string{"admin", "user"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	handler.Me(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if userID, ok := resp["user_id"].(string); !ok || userID != "user-123" {
		t.Errorf("Expected user_id 'user-123', got %v", resp["user_id"])
	}

	if roles, ok := resp["roles"].([]interface{}); !ok || len(roles) != 2 {
		t.Errorf("Expected 2 roles, got %v", resp["roles"])
	}
}

func TestAuthHandler_Me_NoContext(t *testing.T) {
	server, authClient := createMockAuthClient()
	defer server.Close()

	handler := NewAuthHandler(authClient, "localhost", false)

	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	rr := httptest.NewRecorder()

	handler.Me(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if userID, ok := resp["user_id"].(string); !ok || userID != "" {
		t.Errorf("Expected empty user_id, got %v", resp["user_id"])
	}

	if roles, ok := resp["roles"].([]interface{}); !ok || len(roles) != 0 {
		t.Errorf("Expected empty roles array, got %v", resp["roles"])
	}
}

func TestAuthHandler_CookieSettings(t *testing.T) {
	tests := []struct {
		name         string
		cookieDomain string
		cookieSecure bool
	}{
		{
			name:         "Secure cookie",
			cookieDomain: "example.com",
			cookieSecure: true,
		},
		{
			name:         "Insecure cookie for localhost",
			cookieDomain: "localhost",
			cookieSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, authClient := createMockAuthClient()
			defer server.Close()

			handler := NewAuthHandler(authClient, tt.cookieDomain, tt.cookieSecure)

			loginReq := LoginRequest{
				Username: "admin",
				Password: "admin123",
			}
			body, _ := json.Marshal(loginReq)

			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.Login(rr, req)

			cookies := rr.Result().Cookies()
			for _, cookie := range cookies {
				if cookie.Domain != tt.cookieDomain {
					t.Errorf("Expected domain '%s', got '%s'", tt.cookieDomain, cookie.Domain)
				}
				if cookie.Secure != tt.cookieSecure {
					t.Errorf("Expected Secure %v, got %v", tt.cookieSecure, cookie.Secure)
				}
			}
		})
	}
}
