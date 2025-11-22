package authclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      *tokenCache
}

type ValidateHECTokenRequest struct {
	Token string `json:"token"`
}

type ValidateHECTokenResponse struct {
	Valid     bool   `json:"valid"`
	TokenID   string `json:"token_id,omitempty"`
	TokenName string `json:"token_name,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"` // Client tenant for data isolation
}

type tokenCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	valid     bool
	tokenInfo ValidateHECTokenResponse
	expiresAt time.Time
}

func New(baseURL string, timeout time.Duration, cacheTTL time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		cache: &tokenCache{
			entries: make(map[string]*cacheEntry),
			ttl:     cacheTTL,
		},
	}
}

func (c *Client) ValidateHECToken(ctx context.Context, token string) (*ValidateHECTokenResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("auth client not configured")
	}

	// Check cache first
	if cached := c.cache.get(token); cached != nil {
		return cached, nil
	}

	reqBody := ValidateHECTokenRequest{
		Token: token,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/auth/validate-hec", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result ValidateHECTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Cache the result
	c.cache.set(token, &result)

	return &result, nil
}

func (tc *tokenCache) get(token string) *ValidateHECTokenResponse {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	entry, exists := tc.entries[token]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil
	}

	return &entry.tokenInfo
}

func (tc *tokenCache) set(token string, info *ValidateHECTokenResponse) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.entries[token] = &cacheEntry{
		valid:     info.Valid,
		tokenInfo: *info,
		expiresAt: time.Now().Add(tc.ttl),
	}

	// Clean up expired entries periodically
	go tc.cleanup()
}

func (tc *tokenCache) cleanup() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now()
	for token, entry := range tc.entries {
		if now.After(entry.expiresAt) {
			delete(tc.entries, token)
		}
	}
}
