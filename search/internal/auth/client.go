package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ValidateResponse struct {
	Valid  bool     `json:"valid"`
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c *Client) Validate(ctx context.Context, bearerToken string) (*ValidateResponse, error) {
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
	return &vr, nil
}
