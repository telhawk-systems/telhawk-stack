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
	ClientID  string `json:"client_id,omitempty"` // Client for data isolation
}

type tokenCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	maxSize int
}

type cacheEntry struct {
	valid     bool
	tokenInfo ValidateHECTokenResponse
	expiresAt time.Time
}

func newTokenCache(ttl time.Duration, maxSize int) *tokenCache {
	tc := &tokenCache{ttl: ttl, maxSize: maxSize, entries: make(map[string]*cacheEntry)}
	go tc.runCleanup()
	return tc
}

func (tc *tokenCache) runCleanup() {
	ticker := time.NewTicker(tc.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		tc.cleanup()
	}
}

func New(baseURL string, timeout time.Duration, cacheTTL time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		cache: newTokenCache(cacheTTL, 10000),
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

	// Evict an arbitrary entry when at capacity
	if len(tc.entries) >= tc.maxSize {
		for k := range tc.entries {
			delete(tc.entries, k)
			break
		}
	}

	tc.entries[token] = &cacheEntry{
		valid:     info.Valid,
		tokenInfo: *info,
		expiresAt: time.Now().Add(tc.ttl),
	}
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
