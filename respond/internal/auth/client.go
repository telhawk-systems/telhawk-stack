// Package auth provides authentication client for the respond service.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ValidateResponse represents the response from the auth service validation endpoint.
type ValidateResponse struct {
	Valid          bool     `json:"valid"`
	UserID         string   `json:"user_id"`
	Roles          []string `json:"roles"`
	OrganizationID string   `json:"organization_id,omitempty"` // Primary org for data isolation
	ClientID       string   `json:"client_id,omitempty"`       // Primary client for data isolation
}

// UserContext holds authenticated user information including data isolation context.
type UserContext struct {
	UserID         string
	Roles          []string
	OrganizationID string
	ClientID       string
}

// Client provides authentication validation by calling the authenticate service.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a new auth client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Validate validates a bearer token and returns user context including client_id.
func (c *Client) Validate(ctx context.Context, bearerToken string) (*UserContext, error) {
	if bearerToken == "" {
		return nil, fmt.Errorf("missing bearer token")
	}

	body, _ := json.Marshal(map[string]string{"token": bearerToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/auth/validate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth validate returned %d", resp.StatusCode)
	}

	var vr ValidateResponse
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		return nil, err
	}

	if !vr.Valid || vr.UserID == "" {
		return nil, fmt.Errorf("invalid token")
	}

	return &UserContext{
		UserID:         vr.UserID,
		Roles:          vr.Roles,
		OrganizationID: vr.OrganizationID,
		ClientID:       vr.ClientID,
	}, nil
}
