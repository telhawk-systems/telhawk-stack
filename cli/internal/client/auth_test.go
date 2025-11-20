package client

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test JWT token with user_id claim
func createTestJWT(userID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"` + userID + `","exp":9999999999}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))
	return header + "." + payload + "." + signature
}

func TestNewAuthClient(t *testing.T) {
	client := NewAuthClient("http://localhost:8080")

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080", client.baseURL)
	assert.NotNil(t, client.client)
	assert.Equal(t, 10*time.Second, client.client.Timeout)
}

func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/auth/login", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		assert.Equal(t, "testuser", payload["username"])
		assert.Equal(t, "testpass", payload["password"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponse{
			AccessToken:  "access-token-123",
			RefreshToken: "refresh-token-456",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	resp, err := client.Login("testuser", "testpass")

	require.NoError(t, err)
	assert.Equal(t, "access-token-123", resp.AccessToken)
	assert.Equal(t, "refresh-token-456", resp.RefreshToken)
	assert.Equal(t, 3600, resp.ExpiresIn)
	assert.Equal(t, "Bearer", resp.TokenType)
}

func TestLogin_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid credentials"}`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	resp, err := client.Login("baduser", "badpass")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "login failed")
}

func TestLogin_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewAuthClient("http://invalid-host-that-does-not-exist.local:99999")
	resp, err := client.Login("user", "pass")

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestLogin_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	resp, err := client.Login("user", "pass")

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestValidateToken_Valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/auth/validate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "test-token", payload["token"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ValidateResponse{
			Valid:  true,
			UserID: "user-123",
			Roles:  []string{"admin", "user"},
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	resp, err := client.ValidateToken("test-token")

	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "user-123", resp.UserID)
	assert.Equal(t, []string{"admin", "user"}, resp.Roles)
}

func TestValidateToken_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ValidateResponse{
			Valid: false,
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	resp, err := client.ValidateToken("invalid-token")

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Empty(t, resp.UserID)
}

func TestCreateHECToken_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/hec/tokens", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "user-123", r.Header.Get("X-User-ID"))

		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "Test Token", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(HECToken{
			ID:        "token-id-123",
			Token:     "hec-token-abc",
			Name:      "Test Token",
			UserID:    "user-123",
			Enabled:   true,
			CreatedAt: time.Now(),
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	resp, err := client.CreateHECToken(testToken, "Test Token", "")

	require.NoError(t, err)
	assert.Equal(t, "token-id-123", resp.ID)
	assert.Equal(t, "hec-token-abc", resp.Token)
	assert.Equal(t, "Test Token", resp.Name)
	assert.Equal(t, "user-123", resp.UserID)
	assert.True(t, resp.Enabled)
}

func TestCreateHECToken_WithExpiry(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "30d", payload["expires_in"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(HECToken{
			Token:     "hec-token-abc",
			Name:      "Expiring Token",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	resp, err := client.CreateHECToken(testToken, "Expiring Token", "30d")

	require.NoError(t, err)
	assert.False(t, resp.ExpiresAt.IsZero())
}

func TestCreateHECToken_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	resp, err := client.CreateHECToken("bad-token", "Test", "")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create token")
}

func TestListHECTokens_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/hec/tokens", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "user-123", r.Header.Get("X-User-ID"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*HECToken{
			{
				ID:      "token-1",
				Token:   "hec-1",
				Name:    "Token 1",
				Enabled: true,
			},
			{
				ID:      "token-2",
				Token:   "hec-2",
				Name:    "Token 2",
				Enabled: false,
			},
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	tokens, err := client.ListHECTokens(testToken)

	require.NoError(t, err)
	assert.Len(t, tokens, 2)
	assert.Equal(t, "token-1", tokens[0].ID)
	assert.Equal(t, "Token 1", tokens[0].Name)
	assert.True(t, tokens[0].Enabled)
	assert.Equal(t, "token-2", tokens[1].ID)
	assert.False(t, tokens[1].Enabled)
}

func TestListHECTokens_Empty(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*HECToken{})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	tokens, err := client.ListHECTokens(testToken)

	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestListHECTokens_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	tokens, err := client.ListHECTokens("bad-token")

	assert.Error(t, err)
	assert.Nil(t, tokens)
}

func TestRevokeHECToken_Success(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/hec/tokens/revoke", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "user-123", r.Header.Get("X-User-ID"))

		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "hec-token-to-revoke", payload["token"])

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	err := client.RevokeHECToken(testToken, "hec-token-to-revoke")

	assert.NoError(t, err)
}

func TestRevokeHECToken_Failure(t *testing.T) {
	testToken := createTestJWT("user-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"token not found"}`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	err := client.RevokeHECToken(testToken, "nonexistent-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to revoke token")
}

func TestExtractUserIDFromToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "valid JWT with user_id",
			token:    createTestJWT("user-123"),
			expected: "user-123",
		},
		{
			name:     "invalid JWT format",
			token:    "not.a.jwt.token.invalid",
			expected: "",
		},
		{
			name:     "invalid base64",
			token:    "header.!!!invalid-base64!!!.signature",
			expected: "",
		},
		{
			name:     "valid JWT but no user_id claim",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			expected: "",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUserIDFromToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestListUsers_Success(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/users", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*User{
			{
				ID:       "user-1",
				Username: "alice",
				Email:    "alice@example.com",
				Roles:    []string{"admin"},
				Enabled:  true,
			},
			{
				ID:       "user-2",
				Username: "bob",
				Email:    "bob@example.com",
				Roles:    []string{"user"},
				Enabled:  true,
			},
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	users, err := client.ListUsers(testToken)

	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "alice", users[0].Username)
	assert.Equal(t, "bob", users[1].Username)
}

func TestListUsers_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"insufficient permissions"}`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	users, err := client.ListUsers("non-admin-token")

	assert.Error(t, err)
	assert.Nil(t, users)
	assert.Contains(t, err.Error(), "failed to list users")
}

func TestGetUser_Success(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/users/get", r.URL.Path)
		assert.Equal(t, "user-456", r.URL.Query().Get("id"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(User{
			ID:       "user-456",
			Username: "testuser",
			Email:    "test@example.com",
			Roles:    []string{"user"},
			Enabled:  true,
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	user, err := client.GetUser(testToken, "user-456")

	require.NoError(t, err)
	assert.Equal(t, "user-456", user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
}

func TestGetUser_NotFound(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"user not found"}`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	user, err := client.GetUser(testToken, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestCreateUser_Success(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/auth/register", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "newuser", payload["username"])
		assert.Equal(t, "new@example.com", payload["email"])
		assert.Equal(t, "password123", payload["password"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(User{
			ID:       "new-user-id",
			Username: "newuser",
			Email:    "new@example.com",
			Roles:    []string{"user"},
			Enabled:  true,
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	user, err := client.CreateUser(testToken, "newuser", "new@example.com", "password123", []string{"user"})

	require.NoError(t, err)
	assert.Equal(t, "new-user-id", user.ID)
	assert.Equal(t, "newuser", user.Username)
}

func TestCreateUser_Conflict(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"username already exists"}`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	user, err := client.CreateUser(testToken, "existing", "test@example.com", "pass", []string{"user"})

	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUpdateUser_Success(t *testing.T) {
	testToken := createTestJWT("admin-123")
	enabled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/users/update", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "user-789", r.URL.Query().Get("id"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", payload["email"])
		assert.Equal(t, false, payload["enabled"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(User{
			ID:      "user-789",
			Email:   "updated@example.com",
			Enabled: false,
		})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	user, err := client.UpdateUser(testToken, "user-789", "updated@example.com", nil, &enabled)

	require.NoError(t, err)
	assert.Equal(t, "user-789", user.ID)
	assert.Equal(t, "updated@example.com", user.Email)
	assert.False(t, user.Enabled)
}

func TestUpdateUser_PartialUpdate(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		// Only roles should be in payload
		assert.NotContains(t, payload, "email")
		assert.NotContains(t, payload, "enabled")
		assert.Contains(t, payload, "roles")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(User{ID: "user-1"})
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	user, err := client.UpdateUser(testToken, "user-1", "", []string{"admin"}, nil)

	require.NoError(t, err)
	assert.NotNil(t, user)
}

func TestDeleteUser_Success(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/users/delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "user-to-delete", r.URL.Query().Get("id"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	err := client.DeleteUser(testToken, "user-to-delete")

	assert.NoError(t, err)
}

func TestDeleteUser_NotFound(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"user not found"}`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	err := client.DeleteUser(testToken, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete user")
}

func TestResetPassword_Success(t *testing.T) {
	testToken := createTestJWT("admin-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/users/reset-password", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "user-reset", r.URL.Query().Get("id"))

		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "newpassword123", payload["new_password"])

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	err := client.ResetPassword(testToken, "user-reset", "newpassword123")

	assert.NoError(t, err)
}

func TestResetPassword_Unauthorized(t *testing.T) {
	testToken := createTestJWT("regular-user")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"insufficient permissions"}`))
	}))
	defer server.Close()

	client := NewAuthClient(server.URL)
	err := client.ResetPassword(testToken, "other-user", "newpass")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reset password")
}
