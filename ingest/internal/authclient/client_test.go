package authclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	baseURL := "http://localhost:8080"
	timeout := 10 * time.Second
	cacheTTL := 5 * time.Minute

	client := New(baseURL, timeout, cacheTTL)

	if client == nil {
		t.Fatal("New() returned nil")
	}

	if client.baseURL != baseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, baseURL)
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}

	if client.httpClient.Timeout != timeout {
		t.Errorf("httpClient.Timeout = %v, want %v", client.httpClient.Timeout, timeout)
	}

	if client.cache == nil {
		t.Error("cache is nil")
	}

	if client.cache.ttl != cacheTTL {
		t.Errorf("cache.ttl = %v, want %v", client.cache.ttl, cacheTTL)
	}
}

func TestValidateHECToken_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/validate-hec" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var req ValidateHECTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.Token != "valid-token" {
			t.Errorf("unexpected token: %s", req.Token)
		}

		resp := ValidateHECTokenResponse{
			Valid:     true,
			TokenID:   "token-123",
			TokenName: "Test Token",
			UserID:    "user-456",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second, 1*time.Minute)
	ctx := context.Background()

	result, err := client.ValidateHECToken(ctx, "valid-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if !result.Valid {
		t.Error("Expected Valid = true")
	}

	if result.TokenID != "token-123" {
		t.Errorf("TokenID = %q, want %q", result.TokenID, "token-123")
	}

	if result.TokenName != "Test Token" {
		t.Errorf("TokenName = %q, want %q", result.TokenName, "Test Token")
	}

	if result.UserID != "user-456" {
		t.Errorf("UserID = %q, want %q", result.UserID, "user-456")
	}
}

func TestValidateHECToken_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ValidateHECTokenResponse{
			Valid: false,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second, 1*time.Minute)
	ctx := context.Background()

	result, err := client.ValidateHECToken(ctx, "invalid-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if result.Valid {
		t.Error("Expected Valid = false for invalid token")
	}
}

func TestValidateHECToken_NilClient(t *testing.T) {
	var client *Client
	ctx := context.Background()

	_, err := client.ValidateHECToken(ctx, "test-token")
	if err == nil {
		t.Error("ValidateHECToken() with nil client should return error")
	}

	expectedErr := "auth client not configured"
	if err.Error() != expectedErr {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestValidateHECToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second, 1*time.Minute)
	ctx := context.Background()

	// Should still succeed in decoding (even if response is not valid JSON)
	// The actual error handling depends on the server response
	_, err := client.ValidateHECToken(ctx, "test-token")
	if err == nil {
		t.Error("Expected error when server returns 500")
	}
}

func TestValidateHECToken_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		resp := ValidateHECTokenResponse{Valid: true}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second, 1*time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.ValidateHECToken(ctx, "test-token")
	if err == nil {
		t.Error("ValidateHECToken() with cancelled context should return error")
	}
}

func TestValidateHECToken_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		resp := ValidateHECTokenResponse{Valid: true}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Set very short timeout
	client := New(server.URL, 50*time.Millisecond, 1*time.Minute)
	ctx := context.Background()

	_, err := client.ValidateHECToken(ctx, "test-token")
	if err == nil {
		t.Error("ValidateHECToken() should timeout")
	}
}

func TestTokenCache_HitAndMiss(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := ValidateHECTokenResponse{
			Valid:   true,
			TokenID: "token-123",
			UserID:  "user-456",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second, 1*time.Minute)
	ctx := context.Background()

	// First call - cache miss
	result1, err := client.ValidateHECToken(ctx, "cached-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 server call, got %d", callCount)
	}

	// Second call - cache hit
	result2, err := client.ValidateHECToken(ctx, "cached-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 server call (cached), got %d", callCount)
	}

	// Verify cached result matches
	if result1.TokenID != result2.TokenID {
		t.Errorf("Cached result TokenID = %q, want %q", result2.TokenID, result1.TokenID)
	}

	// Third call with different token - cache miss
	_, err = client.ValidateHECToken(ctx, "different-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 server calls, got %d", callCount)
	}
}

func TestTokenCache_Expiration(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := ValidateHECTokenResponse{
			Valid:   true,
			TokenID: "token-123",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Very short TTL for testing
	client := New(server.URL, 5*time.Second, 100*time.Millisecond)
	ctx := context.Background()

	// First call
	_, err := client.ValidateHECToken(ctx, "expiring-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 server call, got %d", callCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Second call - cache should be expired
	_, err = client.ValidateHECToken(ctx, "expiring-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 server calls after expiration, got %d", callCount)
	}
}

func TestTokenCache_ConcurrentAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ValidateHECTokenResponse{
			Valid:   true,
			TokenID: "token-123",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second, 1*time.Minute)
	ctx := context.Background()

	// Run multiple goroutines accessing cache concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				token := "concurrent-token"
				_, err := client.ValidateHECToken(ctx, token)
				if err != nil {
					t.Errorf("Goroutine %d: ValidateHECToken() error = %v", id, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestTokenCache_Cleanup(t *testing.T) {
	cache := &tokenCache{
		entries: make(map[string]*cacheEntry),
		ttl:     1 * time.Minute,
	}

	// Add some entries
	cache.set("token1", &ValidateHECTokenResponse{Valid: true})
	cache.set("token2", &ValidateHECTokenResponse{Valid: true})

	// Manually set one entry to expired
	cache.mu.Lock()
	cache.entries["token1"].expiresAt = time.Now().Add(-1 * time.Hour)
	cache.mu.Unlock()

	// Run cleanup
	cache.cleanup()

	// Check that expired entry was removed
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if _, exists := cache.entries["token1"]; exists {
		t.Error("Expired token1 should have been removed")
	}

	if _, exists := cache.entries["token2"]; !exists {
		t.Error("Valid token2 should still exist")
	}
}

func TestTokenCache_GetNonExistent(t *testing.T) {
	cache := &tokenCache{
		entries: make(map[string]*cacheEntry),
		ttl:     1 * time.Minute,
	}

	result := cache.get("nonexistent")
	if result != nil {
		t.Error("get() for nonexistent token should return nil")
	}
}

func TestTokenCache_GetExpired(t *testing.T) {
	cache := &tokenCache{
		entries: make(map[string]*cacheEntry),
		ttl:     1 * time.Minute,
	}

	// Add entry and immediately expire it
	cache.set("token", &ValidateHECTokenResponse{Valid: true})
	cache.mu.Lock()
	cache.entries["token"].expiresAt = time.Now().Add(-1 * time.Hour)
	cache.mu.Unlock()

	result := cache.get("token")
	if result != nil {
		t.Error("get() for expired token should return nil")
	}
}

func TestValidateHECToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second, 1*time.Minute)
	ctx := context.Background()

	_, err := client.ValidateHECToken(ctx, "test-token")
	if err == nil {
		t.Error("ValidateHECToken() should error on invalid JSON response")
	}
}

func TestValidateHECToken_MalformedURL(t *testing.T) {
	// Client with invalid base URL
	client := New("http://[invalid", 5*time.Second, 1*time.Minute)
	ctx := context.Background()

	_, err := client.ValidateHECToken(ctx, "test-token")
	if err == nil {
		t.Error("ValidateHECToken() should error with malformed URL")
	}
}

func TestTokenCache_CachesInvalidTokens(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := ValidateHECTokenResponse{
			Valid: false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, 5*time.Second, 1*time.Minute)
	ctx := context.Background()

	// First call
	result1, err := client.ValidateHECToken(ctx, "invalid-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if result1.Valid {
		t.Error("Expected Valid = false")
	}

	// Second call - should be cached
	result2, err := client.ValidateHECToken(ctx, "invalid-token")
	if err != nil {
		t.Fatalf("ValidateHECToken() error = %v", err)
	}

	if result2.Valid {
		t.Error("Expected Valid = false")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 server call (invalid tokens should be cached too), got %d", callCount)
	}
}
