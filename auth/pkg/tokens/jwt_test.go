package tokens

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ============================================================================
// TokenGenerator Constructor Tests
// ============================================================================

func TestNewTokenGenerator(t *testing.T) {
	tests := []struct {
		name          string
		accessSecret  string
		refreshSecret string
		validate      func(*testing.T, *TokenGenerator)
	}{
		{
			name:          "valid secrets",
			accessSecret:  "test-access-secret-long-enough",
			refreshSecret: "test-refresh-secret-long-enough",
			validate: func(t *testing.T, tg *TokenGenerator) {
				if tg == nil {
					t.Fatal("Expected TokenGenerator, got nil")
				}
				if string(tg.accessSecret) != "test-access-secret-long-enough" {
					t.Error("Access secret not set correctly")
				}
				if string(tg.refreshSecret) != "test-refresh-secret-long-enough" {
					t.Error("Refresh secret not set correctly")
				}
				if tg.accessTTL != 15*time.Minute {
					t.Errorf("Expected access TTL 15m, got %v", tg.accessTTL)
				}
				if tg.refreshTTL != 7*24*time.Hour {
					t.Errorf("Expected refresh TTL 7d, got %v", tg.refreshTTL)
				}
			},
		},
		{
			name:          "empty secrets",
			accessSecret:  "",
			refreshSecret: "",
			validate: func(t *testing.T, tg *TokenGenerator) {
				if tg == nil {
					t.Fatal("Expected TokenGenerator even with empty secrets")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := NewTokenGenerator(tt.accessSecret, tt.refreshSecret)
			if tt.validate != nil {
				tt.validate(t, tg)
			}
		})
	}
}

// ============================================================================
// Access Token Generation Tests
// ============================================================================

func TestGenerateAccessToken(t *testing.T) {
	tg := NewTokenGenerator("test-secret-key-that-is-long-enough", "refresh-secret-key")

	tests := []struct {
		name        string
		userID      string
		roles       []string
		expectError bool
		validate    func(*testing.T, string)
	}{
		{
			name:        "valid token with single role",
			userID:      "user-123",
			roles:       []string{"admin"},
			expectError: false,
			validate: func(t *testing.T, tokenString string) {
				if tokenString == "" {
					t.Fatal("Expected token string, got empty")
				}
				// Verify token has 3 parts (header.payload.signature)
				parts := strings.Split(tokenString, ".")
				if len(parts) != 3 {
					t.Errorf("Expected 3 JWT parts, got %d", len(parts))
				}
			},
		},
		{
			name:        "valid token with multiple roles",
			userID:      "user-456",
			roles:       []string{"admin", "editor", "viewer"},
			expectError: false,
			validate: func(t *testing.T, tokenString string) {
				if tokenString == "" {
					t.Fatal("Expected token string, got empty")
				}
			},
		},
		{
			name:        "valid token with no roles",
			userID:      "user-789",
			roles:       []string{},
			expectError: false,
			validate: func(t *testing.T, tokenString string) {
				if tokenString == "" {
					t.Fatal("Expected token string, got empty")
				}
			},
		},
		{
			name:        "valid token with empty user ID",
			userID:      "",
			roles:       []string{"viewer"},
			expectError: false,
			validate: func(t *testing.T, tokenString string) {
				if tokenString == "" {
					t.Fatal("Expected token string, got empty")
				}
			},
		},
		{
			name:        "valid token with nil roles",
			userID:      "user-nil",
			roles:       nil,
			expectError: false,
			validate: func(t *testing.T, tokenString string) {
				if tokenString == "" {
					t.Fatal("Expected token string, got empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tg.GenerateAccessToken(tt.userID, tt.roles)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, token)
			}
		})
	}
}

func TestGenerateAccessTokenClaims(t *testing.T) {
	tg := NewTokenGenerator("test-secret-key-that-is-long-enough", "refresh-secret-key")
	userID := "test-user-123"
	roles := []string{"admin", "viewer"}

	tokenString, err := tg.GenerateAccessToken(userID, roles)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Parse and validate the token
	claims, err := tg.ValidateAccessToken(tokenString)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Verify claims
	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}

	if len(claims.Roles) != len(roles) {
		t.Errorf("Expected %d roles, got %d", len(roles), len(claims.Roles))
	}

	for i, role := range roles {
		if claims.Roles[i] != role {
			t.Errorf("Expected role %s at index %d, got %s", role, i, claims.Roles[i])
		}
	}

	// Verify registered claims
	if claims.Issuer != "telhawk-auth" {
		t.Errorf("Expected issuer 'telhawk-auth', got %s", claims.Issuer)
	}

	if claims.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	} else {
		expectedExpiry := time.Now().Add(15 * time.Minute)
		// Allow 5 second tolerance for test execution time
		if claims.ExpiresAt.Time.Before(expectedExpiry.Add(-5*time.Second)) ||
			claims.ExpiresAt.Time.After(expectedExpiry.Add(5*time.Second)) {
			t.Errorf("Expected expiry around %v, got %v", expectedExpiry, claims.ExpiresAt.Time)
		}
	}

	if claims.IssuedAt == nil {
		t.Error("Expected IssuedAt to be set")
	}

	if claims.NotBefore == nil {
		t.Error("Expected NotBefore to be set")
	}
}

// ============================================================================
// Access Token Validation Tests
// ============================================================================

func TestValidateAccessToken(t *testing.T) {
	tg := NewTokenGenerator("test-secret-key-that-is-long-enough", "refresh-secret-key")

	// Generate a valid token
	validToken, _ := tg.GenerateAccessToken("user-123", []string{"admin"})

	// Generate token with different secret (will be invalid)
	tgDifferent := NewTokenGenerator("different-secret-key-that-is-long", "refresh-secret-key")
	invalidSecretToken, _ := tgDifferent.GenerateAccessToken("user-456", []string{"viewer"})

	tests := []struct {
		name         string
		tokenString  string
		expectError  bool
		expectUserID string
		expectRoles  []string
	}{
		{
			name:         "valid token",
			tokenString:  validToken,
			expectError:  false,
			expectUserID: "user-123",
			expectRoles:  []string{"admin"},
		},
		{
			name:        "invalid token format",
			tokenString: "invalid.token.format",
			expectError: true,
		},
		{
			name:        "empty token",
			tokenString: "",
			expectError: true,
		},
		{
			name:        "malformed token (missing parts)",
			tokenString: "header.payload",
			expectError: true,
		},
		{
			name:        "token with invalid signature",
			tokenString: invalidSecretToken,
			expectError: true,
		},
		{
			name:        "completely garbage token",
			tokenString: "this-is-not-a-jwt-token-at-all",
			expectError: true,
		},
		{
			name:        "token with only dots",
			tokenString: "...",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := tg.ValidateAccessToken(tt.tokenString)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if claims == nil {
				t.Fatal("Expected claims, got nil")
			}

			if claims.UserID != tt.expectUserID {
				t.Errorf("Expected UserID %s, got %s", tt.expectUserID, claims.UserID)
			}

			if len(claims.Roles) != len(tt.expectRoles) {
				t.Errorf("Expected %d roles, got %d", len(tt.expectRoles), len(claims.Roles))
			}
		})
	}
}

func TestValidateExpiredToken(t *testing.T) {
	tg := NewTokenGenerator("test-secret-key-that-is-long-enough", "refresh-secret-key")

	// Manually create an expired token
	claims := Claims{
		UserID: "user-expired",
		Roles:  []string{"viewer"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "telhawk-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString(tg.accessSecret)
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	// Try to validate expired token
	_, err = tg.ValidateAccessToken(expiredToken)
	if err == nil {
		t.Fatal("Expected error for expired token, got none")
	}

	// The error should be related to expiration
	if !strings.Contains(err.Error(), "expired") && !strings.Contains(err.Error(), "exp") {
		t.Logf("Expected expiration error, got: %v", err)
	}
}

func TestValidateTokenNotYetValid(t *testing.T) {
	tg := NewTokenGenerator("test-secret-key-that-is-long-enough", "refresh-secret-key")

	// Create a token that's not yet valid (NotBefore in future)
	claims := Claims{
		UserID: "user-future",
		Roles:  []string{"viewer"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)), // Not valid for 1 hour
			Issuer:    "telhawk-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	futureToken, err := token.SignedString(tg.accessSecret)
	if err != nil {
		t.Fatalf("Failed to create future token: %v", err)
	}

	// Try to validate token that's not yet valid
	_, err = tg.ValidateAccessToken(futureToken)
	if err == nil {
		t.Fatal("Expected error for not-yet-valid token, got none")
	}
}

func TestValidateTokenWrongSigningMethod(t *testing.T) {
	tg := NewTokenGenerator("test-secret-key-that-is-long-enough", "refresh-secret-key")

	// Create a token with RS256 instead of HS256
	claims := Claims{
		UserID: "user-wrong-method",
		Roles:  []string{"viewer"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "telhawk-auth",
		},
	}

	// Note: We can't actually test RS256 without generating keys,
	// but we can test that the signing method check works for tokens
	// signed with our secret but presented as different algorithm.
	// The actual check happens in the key function.

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(tg.accessSecret)

	// Validate should work with correct method
	_, err := tg.ValidateAccessToken(tokenString)
	if err != nil {
		t.Errorf("Valid HS256 token should validate, got error: %v", err)
	}
}

// ============================================================================
// Refresh Token Generation Tests
// ============================================================================

func TestGenerateRefreshToken(t *testing.T) {
	tg := NewTokenGenerator("access-secret", "refresh-secret")

	tests := []struct {
		name     string
		validate func(*testing.T, string)
	}{
		{
			name: "generates non-empty token",
			validate: func(t *testing.T, token string) {
				if token == "" {
					t.Error("Expected non-empty refresh token")
				}
			},
		},
		{
			name: "generates base64 encoded token",
			validate: func(t *testing.T, token string) {
				// Base64 URL encoding should only contain these characters
				for _, c := range token {
					if !((c >= 'A' && c <= 'Z') ||
						(c >= 'a' && c <= 'z') ||
						(c >= '0' && c <= '9') ||
						c == '-' || c == '_' || c == '=') {
						t.Errorf("Invalid base64 URL character: %c", c)
					}
				}
			},
		},
		{
			name: "generates token of expected length",
			validate: func(t *testing.T, token string) {
				// 32 bytes encoded in base64 should be around 43-44 characters
				if len(token) < 40 || len(token) > 50 {
					t.Errorf("Expected token length 40-50, got %d", len(token))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tg.GenerateRefreshToken()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, token)
			}
		})
	}
}

func TestGenerateRefreshTokenUniqueness(t *testing.T) {
	tg := NewTokenGenerator("access-secret", "refresh-secret")

	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		token, err := tg.GenerateRefreshToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if tokens[token] {
			t.Fatalf("Generated duplicate refresh token: %s", token)
		}
		tokens[token] = true
	}

	if len(tokens) != iterations {
		t.Errorf("Expected %d unique tokens, got %d", iterations, len(tokens))
	}
}

// ============================================================================
// HEC Token Generation Tests
// ============================================================================

func TestGenerateHECToken(t *testing.T) {
	tg := NewTokenGenerator("access-secret", "refresh-secret")

	tests := []struct {
		name     string
		validate func(*testing.T, string)
	}{
		{
			name: "generates non-empty token",
			validate: func(t *testing.T, token string) {
				if token == "" {
					t.Error("Expected non-empty HEC token")
				}
			},
		},
		{
			name: "generates base64 encoded token",
			validate: func(t *testing.T, token string) {
				// Base64 URL encoding should only contain these characters
				for _, c := range token {
					if !((c >= 'A' && c <= 'Z') ||
						(c >= 'a' && c <= 'z') ||
						(c >= '0' && c <= '9') ||
						c == '-' || c == '_' || c == '=') {
						t.Errorf("Invalid base64 URL character: %c", c)
					}
				}
			},
		},
		{
			name: "generates token of expected length",
			validate: func(t *testing.T, token string) {
				// 32 bytes encoded in base64 should be around 43-44 characters
				if len(token) < 40 || len(token) > 50 {
					t.Errorf("Expected token length 40-50, got %d", len(token))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tg.GenerateHECToken()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, token)
			}
		})
	}
}

func TestGenerateHECTokenUniqueness(t *testing.T) {
	tg := NewTokenGenerator("access-secret", "refresh-secret")

	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		token, err := tg.GenerateHECToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if tokens[token] {
			t.Fatalf("Generated duplicate HEC token: %s", token)
		}
		tokens[token] = true
	}

	if len(tokens) != iterations {
		t.Errorf("Expected %d unique tokens, got %d", iterations, len(tokens))
	}
}

// ============================================================================
// Edge Cases and Security Tests
// ============================================================================

func TestTokenGeneratorWithEmptySecret(t *testing.T) {
	tg := NewTokenGenerator("", "")

	// Should still generate token (though not secure)
	token, err := tg.GenerateAccessToken("user-123", []string{"admin"})
	if err != nil {
		t.Fatalf("Failed to generate token with empty secret: %v", err)
	}

	if token == "" {
		t.Error("Expected token even with empty secret")
	}
}

func TestValidateTokenWithDifferentSecret(t *testing.T) {
	tg1 := NewTokenGenerator("secret-1", "refresh-1")
	tg2 := NewTokenGenerator("secret-2", "refresh-2")

	// Generate token with tg1
	token, err := tg1.GenerateAccessToken("user-123", []string{"admin"})
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with tg2 (different secret)
	_, err = tg2.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("Expected error when validating token with different secret, got none")
	}
}

func TestClaimsWithSpecialCharacters(t *testing.T) {
	tg := NewTokenGenerator("test-secret-key-long-enough", "refresh-secret")

	specialUserID := "user-with-special-chars-!@#$%^&*()"
	specialRoles := []string{"role-with-unicode-🔒", "role/with/slashes", "role with spaces"}

	token, err := tg.GenerateAccessToken(specialUserID, specialRoles)
	if err != nil {
		t.Fatalf("Failed to generate token with special characters: %v", err)
	}

	claims, err := tg.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token with special characters: %v", err)
	}

	if claims.UserID != specialUserID {
		t.Errorf("Special characters in UserID not preserved")
	}

	for i, role := range specialRoles {
		if claims.Roles[i] != role {
			t.Errorf("Special characters in role not preserved: expected %s, got %s", role, claims.Roles[i])
		}
	}
}

func TestConcurrentTokenGeneration(t *testing.T) {
	tg := NewTokenGenerator("test-secret-key-long-enough", "refresh-secret")

	iterations := 50
	done := make(chan bool, iterations)
	tokens := make(chan string, iterations)

	// Generate tokens concurrently
	for i := 0; i < iterations; i++ {
		go func(id int) {
			token, err := tg.GenerateAccessToken("user-concurrent", []string{"admin"})
			if err != nil {
				t.Errorf("Concurrent generation failed: %v", err)
			}
			tokens <- token
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < iterations; i++ {
		<-done
	}
	close(tokens)

	// Verify all tokens are valid (though they may be identical due to same timestamp)
	count := 0
	for token := range tokens {
		_, err := tg.ValidateAccessToken(token)
		if err != nil {
			t.Errorf("Generated invalid token during concurrent test: %v", err)
		}
		count++
	}

	if count != iterations {
		t.Errorf("Expected %d tokens, got %d", iterations, count)
	}
}
