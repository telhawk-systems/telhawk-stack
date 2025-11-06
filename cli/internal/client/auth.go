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

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

func (c *AuthClient) ListUsers(accessToken string) ([]*User, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/users", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list users: %s", string(bodyBytes))
	}

	var users []*User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	return users, nil
}

func (c *AuthClient) GetUser(accessToken, userID string) (*User, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/users/get?id="+userID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user: %s", string(bodyBytes))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *AuthClient) CreateUser(accessToken, username, email, password string, roles []string) (*User, error) {
	payload := map[string]interface{}{
		"username": username,
		"email":    email,
		"password": password,
		"roles":    roles,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/auth/register", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create user: %s", string(bodyBytes))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *AuthClient) UpdateUser(accessToken, userID, email string, roles []string, enabled *bool) (*User, error) {
	payload := make(map[string]interface{})
	if email != "" {
		payload["email"] = email
	}
	if roles != nil {
		payload["roles"] = roles
	}
	if enabled != nil {
		payload["enabled"] = *enabled
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", c.baseURL+"/api/v1/users/update?id="+userID, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update user: %s", string(bodyBytes))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *AuthClient) DeleteUser(accessToken, userID string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+"/api/v1/users/delete?id="+userID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user: %s", string(bodyBytes))
	}

	return nil
}

func (c *AuthClient) ResetPassword(accessToken, userID, newPassword string) error {
	payload := map[string]string{
		"new_password": newPassword,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/users/reset-password?id="+userID, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to reset password: %s", string(bodyBytes))
	}

	return nil
}
