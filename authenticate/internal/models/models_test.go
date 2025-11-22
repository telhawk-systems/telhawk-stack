package models

import (
	"testing"
	"time"
)

// ============================================================================
// User Tests
// ============================================================================

func TestUser_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		user     *User
		expected bool
	}{
		{
			name: "active user",
			user: &User{
				ID:         "user-1",
				Username:   "active",
				DisabledAt: nil,
				DeletedAt:  nil,
			},
			expected: true,
		},
		{
			name: "disabled user",
			user: &User{
				ID:         "user-2",
				Username:   "disabled",
				DisabledAt: &now,
				DeletedAt:  nil,
			},
			expected: false,
		},
		{
			name: "deleted user",
			user: &User{
				ID:         "user-3",
				Username:   "deleted",
				DisabledAt: nil,
				DeletedAt:  &now,
			},
			expected: false,
		},
		{
			name: "disabled and deleted user",
			user: &User{
				ID:         "user-4",
				Username:   "disabled-deleted",
				DisabledAt: &now,
				DeletedAt:  &now,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.IsActive()
			if result != tt.expected {
				t.Errorf("Expected IsActive() = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestUser_ToResponse(t *testing.T) {
	now := time.Now()
	disabledTime := now.Add(-1 * time.Hour)

	tests := []struct {
		name            string
		user            *User
		expectedEnabled bool
	}{
		{
			name: "active user",
			user: &User{
				ID:         "user-1",
				Username:   "active",
				Email:      "active@example.com",
				Roles:      []string{"viewer"},
				CreatedAt:  now,
				DisabledAt: nil,
				DeletedAt:  nil,
			},
			expectedEnabled: true,
		},
		{
			name: "disabled user",
			user: &User{
				ID:         "user-2",
				Username:   "disabled",
				Email:      "disabled@example.com",
				Roles:      []string{"admin"},
				CreatedAt:  now,
				DisabledAt: &disabledTime,
				DeletedAt:  nil,
			},
			expectedEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.user.ToResponse()

			if response == nil {
				t.Fatal("ToResponse() returned nil")
			}

			if response.ID != tt.user.ID {
				t.Errorf("Expected ID %s, got %s", tt.user.ID, response.ID)
			}

			if response.Username != tt.user.Username {
				t.Errorf("Expected username %s, got %s", tt.user.Username, response.Username)
			}

			if response.Email != tt.user.Email {
				t.Errorf("Expected email %s, got %s", tt.user.Email, response.Email)
			}

			if len(response.Roles) != len(tt.user.Roles) {
				t.Errorf("Expected %d roles, got %d", len(tt.user.Roles), len(response.Roles))
			}

			if response.Enabled != tt.expectedEnabled {
				t.Errorf("Expected Enabled = %v, got %v", tt.expectedEnabled, response.Enabled)
			}

			if !response.CreatedAt.Equal(tt.user.CreatedAt) {
				t.Errorf("Expected CreatedAt %v, got %v", tt.user.CreatedAt, response.CreatedAt)
			}
		})
	}
}

// ============================================================================
// HECToken Tests
// ============================================================================

func TestHECToken_IsActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name     string
		token    *HECToken
		expected bool
	}{
		{
			name: "active token without expiry",
			token: &HECToken{
				ID:         "token-1",
				Token:      "valid-token",
				DisabledAt: nil,
				RevokedAt:  nil,
				ExpiresAt:  nil,
			},
			expected: true,
		},
		{
			name: "active token with future expiry",
			token: &HECToken{
				ID:         "token-2",
				Token:      "valid-token",
				DisabledAt: nil,
				RevokedAt:  nil,
				ExpiresAt:  &future,
			},
			expected: true,
		},
		{
			name: "disabled token",
			token: &HECToken{
				ID:         "token-3",
				Token:      "disabled-token",
				DisabledAt: &now,
				RevokedAt:  nil,
				ExpiresAt:  nil,
			},
			expected: false,
		},
		{
			name: "revoked token",
			token: &HECToken{
				ID:         "token-4",
				Token:      "revoked-token",
				DisabledAt: nil,
				RevokedAt:  &now,
				ExpiresAt:  nil,
			},
			expected: false,
		},
		{
			name: "expired token",
			token: &HECToken{
				ID:         "token-5",
				Token:      "expired-token",
				DisabledAt: nil,
				RevokedAt:  nil,
				ExpiresAt:  &past,
			},
			expected: false,
		},
		{
			name: "disabled and revoked token",
			token: &HECToken{
				ID:         "token-6",
				Token:      "disabled-revoked-token",
				DisabledAt: &now,
				RevokedAt:  &now,
				ExpiresAt:  nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsActive()
			if result != tt.expected {
				t.Errorf("Expected IsActive() = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHECToken_ToResponse(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)

	token := &HECToken{
		ID:        "token-1",
		Token:     "full-token-value-12345",
		Name:      "Test Token",
		UserID:    "user-123",
		CreatedAt: now,
		ExpiresAt: &future,
	}

	response := token.ToResponse()

	if response == nil {
		t.Fatal("ToResponse() returned nil")
	}

	if response.ID != token.ID {
		t.Errorf("Expected ID %s, got %s", token.ID, response.ID)
	}

	if response.Token != token.Token {
		t.Errorf("Expected full token %s, got %s", token.Token, response.Token)
	}

	if response.Name != token.Name {
		t.Errorf("Expected name %s, got %s", token.Name, response.Name)
	}

	if response.UserID != token.UserID {
		t.Errorf("Expected UserID %s, got %s", token.UserID, response.UserID)
	}

	if !response.Enabled {
		t.Error("Expected Enabled = true for active token")
	}

	if response.Username != "" {
		t.Errorf("Expected empty username, got %s", response.Username)
	}
}

func TestHECToken_ToMaskedResponse(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		token         *HECToken
		expectedToken string
	}{
		{
			name: "long token is masked",
			token: &HECToken{
				ID:        "token-1",
				Token:     "abcdefgh123456789ijklmnop",
				Name:      "Test Token",
				UserID:    "user-123",
				CreatedAt: now,
			},
			expectedToken: "abcdefgh...ijklmnop",
		},
		{
			name: "short token is not masked",
			token: &HECToken{
				ID:        "token-2",
				Token:     "shorttoken123",
				Name:      "Short Token",
				UserID:    "user-456",
				CreatedAt: now,
			},
			expectedToken: "shorttoken123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.token.ToMaskedResponse()

			if response == nil {
				t.Fatal("ToMaskedResponse() returned nil")
			}

			if response.Token != tt.expectedToken {
				t.Errorf("Expected masked token %s, got %s", tt.expectedToken, response.Token)
			}

			if response.ID != tt.token.ID {
				t.Errorf("Expected ID %s, got %s", tt.token.ID, response.ID)
			}

			if response.Username != "" {
				t.Errorf("Expected empty username, got %s", response.Username)
			}
		})
	}
}

func TestHECToken_ToMaskedResponseWithUsername(t *testing.T) {
	now := time.Now()

	token := &HECToken{
		ID:        "token-1",
		Token:     "abcdefgh123456789ijklmnop",
		Name:      "Test Token",
		UserID:    "user-123",
		CreatedAt: now,
	}

	username := "testuser"
	response := token.ToMaskedResponseWithUsername(username)

	if response == nil {
		t.Fatal("ToMaskedResponseWithUsername() returned nil")
	}

	if response.Token != "abcdefgh...ijklmnop" {
		t.Errorf("Expected masked token, got %s", response.Token)
	}

	if response.Username != username {
		t.Errorf("Expected username %s, got %s", username, response.Username)
	}

	if response.ID != token.ID {
		t.Errorf("Expected ID %s, got %s", token.ID, response.ID)
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "long token is masked",
			token:    "abcdefgh123456789ijklmnop",
			expected: "abcdefgh...ijklmnop",
		},
		{
			name:     "short token not masked",
			token:    "short",
			expected: "short",
		},
		{
			name:     "exactly 16 chars not masked",
			token:    "1234567890123456",
			expected: "1234567890123456",
		},
		{
			name:     "17 chars is masked",
			token:    "12345678901234567",
			expected: "12345678...01234567",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "",
		},
		{
			name:     "very long token",
			token:    "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			expected: "abcdefgh...STUVWXYZ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskToken(tt.token)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Session Tests
// ============================================================================

func TestSession_IsActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name     string
		session  *Session
		expected bool
	}{
		{
			name: "active session",
			session: &Session{
				ID:        "session-1",
				UserID:    "user-123",
				ExpiresAt: future,
				RevokedAt: nil,
			},
			expected: true,
		},
		{
			name: "expired session",
			session: &Session{
				ID:        "session-2",
				UserID:    "user-123",
				ExpiresAt: past,
				RevokedAt: nil,
			},
			expected: false,
		},
		{
			name: "revoked session",
			session: &Session{
				ID:        "session-3",
				UserID:    "user-123",
				ExpiresAt: future,
				RevokedAt: &now,
			},
			expected: false,
		},
		{
			name: "revoked and expired session",
			session: &Session{
				ID:        "session-4",
				UserID:    "user-123",
				ExpiresAt: past,
				RevokedAt: &now,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session.IsActive()
			if result != tt.expected {
				t.Errorf("Expected IsActive() = %v, got %v", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Audit Tests
// ============================================================================

func TestShouldForwardToIngest(t *testing.T) {
	tests := []struct {
		action   string
		expected bool
	}{
		{ActionLogin, true},
		{ActionLogout, true},
		{ActionUserCreate, true},
		{ActionUserDelete, true},
		{ActionPasswordReset, true},
		{ActionHECTokenCreate, true},
		{ActionHECTokenRevoke, true},
		{ActionTokenValidate, false},    // Too frequent
		{ActionTokenRefresh, false},     // Too frequent
		{ActionHECTokenValidate, false}, // Would cause loop
		{"custom_action", true},         // Unknown actions forwarded by default
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			result := ShouldForwardToIngest(tt.action)
			if result != tt.expected {
				t.Errorf("Action %s: expected %v, got %v", tt.action, tt.expected, result)
			}
		})
	}
}
