package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AuthClient struct {
	baseURL string
	client  *http.Client
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type ValidateResponse struct {
	Valid  bool     `json:"valid"`
	UserID string   `json:"user_id,omitempty"`
	Roles  []string `json:"roles,omitempty"`
}

type HECToken struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	Name      string    `json:"name"`
	UserID    string    `json:"user_id"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *AuthClient) Login(username, password string) (*LoginResponse, error) {
	payload := map[string]string{
		"username": username,
		"password": password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Post(
		c.baseURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed: %s", string(bodyBytes))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, err
	}

	return &loginResp, nil
}

func (c *AuthClient) ValidateToken(token string) (*ValidateResponse, error) {
	payload := map[string]string{
		"token": token,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Post(
		c.baseURL+"/api/v1/auth/validate",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var validateResp ValidateResponse
	if err := json.NewDecoder(resp.Body).Decode(&validateResp); err != nil {
		return nil, err
	}

	return &validateResp, nil
}

func (c *AuthClient) CreateHECToken(accessToken, name, expires string) (*HECToken, error) {
	// Placeholder - will be implemented when HEC token endpoint is added to auth service
	return &HECToken{
		Token:     "mock-hec-token-" + name,
		Name:      name,
		Enabled:   true,
		CreatedAt: time.Now(),
	}, nil
}

func (c *AuthClient) ListHECTokens(accessToken string) ([]*HECToken, error) {
	// Placeholder - will be implemented when HEC token endpoint is added to auth service
	return []*HECToken{}, nil
}

func (c *AuthClient) RevokeHECToken(accessToken, token string) error {
	// Placeholder - will be implemented when HEC token endpoint is added to auth service
	return nil
}
