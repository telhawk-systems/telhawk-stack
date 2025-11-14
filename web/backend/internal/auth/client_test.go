package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	baseURL := "http://localhost:8080"
	client := NewClient(baseURL)

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL '%s', got '%s'", baseURL, client.baseURL)
	}

	if client.httpClient == nil {
		t.Fatal("Expected httpClient to be initialized")
	}

	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.httpClient.Timeout)
	}
}

func TestClient_Login_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("Expected path /api/v1/auth/login, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req.Username != "admin" {
			t.Errorf("Expected username 'admin', got '%s'", req.Username)
		}
		if req.Password != "admin123" {
			t.Errorf("Expected password 'admin123', got '%s'", req.Password)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponse{
			AccessToken:  "access-token-123",
			RefreshToken: "refresh-token-456",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.Login("admin", "admin123")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.AccessToken != "access-token-123" {
		t.Errorf("Expected AccessToken 'access-token-123', got '%s'", resp.AccessToken)
	}
	if resp.RefreshToken != "refresh-token-456" {
		t.Errorf("Expected RefreshToken 'refresh-token-456', got '%s'", resp.RefreshToken)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("Expected ExpiresIn 3600, got %d", resp.ExpiresIn)
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("Expected TokenType 'Bearer', got '%s'", resp.TokenType)
	}
}

func TestClient_Login_InvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid credentials",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.Login("admin", "wrongpassword")

	if err == nil {
		t.Fatal("Expected error for invalid credentials")
	}

	if resp != nil {
		t.Error("Expected nil response for failed login")
	}

	expectedErrMsg := "login failed: 401"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestClient_Login_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.Login("admin", "admin123")

	if err == nil {
		t.Fatal("Expected error for server error")
	}

	if resp != nil {
		t.Error("Expected nil response for server error")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to contain status code 500, got '%s'", err.Error())
	}
}

func TestClient_Login_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.Login("admin", "admin123")

	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}

	if resp != nil {
		t.Error("Expected nil response for malformed JSON")
	}
}

func TestClient_Login_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewClient("http://localhost:99999")
	resp, err := client.Login("admin", "admin123")

	if err == nil {
		t.Fatal("Expected error for network failure")
	}

	if resp != nil {
		t.Error("Expected nil response for network error")
	}
}

func TestClient_ValidateToken_Valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/validate" {
			t.Errorf("Expected path /api/v1/auth/validate, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req["token"] != "valid-token" {
			t.Errorf("Expected token 'valid-token', got '%s'", req["token"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ValidateResponse{
			Valid:  true,
			UserID: "user-123",
			Roles:  []string{"admin", "user"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ValidateToken("valid-token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !resp.Valid {
		t.Error("Expected Valid to be true")
	}
	if resp.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", resp.UserID)
	}
	if len(resp.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(resp.Roles))
	}
}

func TestClient_ValidateToken_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ValidateResponse{
			Valid: false,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ValidateToken("invalid-token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Valid {
		t.Error("Expected Valid to be false")
	}
}

func TestClient_ValidateToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ValidateToken("some-token")

	if err != nil {
		t.Fatalf("Expected no error (returns invalid), got %v", err)
	}

	if resp.Valid {
		t.Error("Expected Valid to be false for server error")
	}
}

func TestClient_ValidateToken_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ValidateToken("some-token")

	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}

	if resp != nil {
		t.Error("Expected nil response for malformed JSON")
	}
}

func TestClient_ValidateToken_NetworkError(t *testing.T) {
	client := NewClient("http://localhost:99999")
	resp, err := client.ValidateToken("some-token")

	if err == nil {
		t.Fatal("Expected error for network failure")
	}

	if resp != nil {
		t.Error("Expected nil response for network error")
	}
}

func TestClient_RefreshToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/refresh" {
			t.Errorf("Expected path /api/v1/auth/refresh, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		var req RefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.RefreshToken != "refresh-token-123" {
			t.Errorf("Expected RefreshToken 'refresh-token-123', got '%s'", req.RefreshToken)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponse{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.RefreshToken("refresh-token-123")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.AccessToken != "new-access-token" {
		t.Errorf("Expected AccessToken 'new-access-token', got '%s'", resp.AccessToken)
	}
	if resp.RefreshToken != "new-refresh-token" {
		t.Errorf("Expected RefreshToken 'new-refresh-token', got '%s'", resp.RefreshToken)
	}
}

func TestClient_RefreshToken_Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Refresh token expired",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.RefreshToken("expired-token")

	if err == nil {
		t.Fatal("Expected error for expired token")
	}

	if resp != nil {
		t.Error("Expected nil response for expired token")
	}

	if !strings.Contains(err.Error(), "refresh failed: 401") {
		t.Errorf("Expected error to contain 'refresh failed: 401', got '%s'", err.Error())
	}
}

func TestClient_RefreshToken_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.RefreshToken("some-token")

	if err == nil {
		t.Fatal("Expected error for malformed JSON")
	}

	if resp != nil {
		t.Error("Expected nil response for malformed JSON")
	}
}

func TestClient_RefreshToken_NetworkError(t *testing.T) {
	client := NewClient("http://localhost:99999")
	resp, err := client.RefreshToken("some-token")

	if err == nil {
		t.Fatal("Expected error for network failure")
	}

	if resp != nil {
		t.Error("Expected nil response for network error")
	}
}

func TestClient_RevokeToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/revoke" {
			t.Errorf("Expected path /api/v1/auth/revoke, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req["token"] != "token-to-revoke" {
			t.Errorf("Expected token 'token-to-revoke', got '%s'", req["token"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.RevokeToken("token-to-revoke")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClient_RevokeToken_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.RevokeToken("some-token")

	if err != nil {
		t.Fatalf("Expected no error for 204 status, got %v", err)
	}
}

func TestClient_RevokeToken_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid token",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.RevokeToken("invalid-token")

	if err == nil {
		t.Fatal("Expected error for invalid token")
	}

	if !strings.Contains(err.Error(), "revoke failed: 400") {
		t.Errorf("Expected error to contain 'revoke failed: 400', got '%s'", err.Error())
	}
}

func TestClient_RevokeToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.RevokeToken("some-token")

	if err == nil {
		t.Fatal("Expected error for server error")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to contain status code 500, got '%s'", err.Error())
	}
}

func TestClient_RevokeToken_NetworkError(t *testing.T) {
	client := NewClient("http://localhost:99999")
	err := client.RevokeToken("some-token")

	if err == nil {
		t.Fatal("Expected error for network failure")
	}
}

func TestClient_Timeout(t *testing.T) {
	// Create a server that delays longer than client timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second) // Client timeout is 10s
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test that timeout works for each method
	t.Run("Login timeout", func(t *testing.T) {
		_, err := client.Login("admin", "admin123")
		if err == nil {
			t.Error("Expected timeout error for Login")
		}
	})

	t.Run("ValidateToken timeout", func(t *testing.T) {
		_, err := client.ValidateToken("some-token")
		if err == nil {
			t.Error("Expected timeout error for ValidateToken")
		}
	})

	t.Run("RefreshToken timeout", func(t *testing.T) {
		_, err := client.RefreshToken("some-token")
		if err == nil {
			t.Error("Expected timeout error for RefreshToken")
		}
	})

	t.Run("RevokeToken timeout", func(t *testing.T) {
		err := client.RevokeToken("some-token")
		if err == nil {
			t.Error("Expected timeout error for RevokeToken")
		}
	})
}

func TestClient_AllMethods_Integration(t *testing.T) {
	// Integration test that exercises full flow
	callLog := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callLog = append(callLog, r.Method+" "+r.URL.Path)

		switch r.URL.Path {
		case "/api/v1/auth/login":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(LoginResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			})

		case "/api/v1/auth/validate":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ValidateResponse{
				Valid:  true,
				UserID: "user-123",
				Roles:  []string{"admin"},
			})

		case "/api/v1/auth/refresh":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(LoginResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			})

		case "/api/v1/auth/revoke":
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// 1. Login
	loginResp, err := client.Login("admin", "admin123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loginResp.AccessToken != "access-token" {
		t.Error("Login did not return expected access token")
	}

	// 2. Validate token
	validateResp, err := client.ValidateToken(loginResp.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if !validateResp.Valid {
		t.Error("Token should be valid")
	}

	// 3. Refresh token
	refreshResp, err := client.RefreshToken(loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if refreshResp.AccessToken != "new-access-token" {
		t.Error("Refresh did not return new access token")
	}

	// 4. Revoke token
	err = client.RevokeToken(refreshResp.RefreshToken)
	if err != nil {
		t.Fatalf("RevokeToken failed: %v", err)
	}

	// Verify all endpoints were called
	expectedCalls := []string{
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/validate",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/revoke",
	}

	if len(callLog) != len(expectedCalls) {
		t.Errorf("Expected %d calls, got %d", len(expectedCalls), len(callLog))
	}

	for i, expected := range expectedCalls {
		if i >= len(callLog) {
			t.Errorf("Missing expected call: %s", expected)
			continue
		}
		if callLog[i] != expected {
			t.Errorf("Call %d: expected '%s', got '%s'", i, expected, callLog[i])
		}
	}
}
