package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPRulesClient implements RulesClient using HTTP
type HTTPRulesClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPRulesClient creates a new HTTP rules client
func NewHTTPRulesClient(baseURL string) *HTTPRulesClient {
	return &HTTPRulesClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ListSchemas fetches all active detection schemas from the Rules service
func (c *HTTPRulesClient) ListSchemas(ctx context.Context) ([]*DetectionSchema, error) {
	// Fetch schemas without disabled or hidden ones
	url := fmt.Sprintf("%s/api/v1/schemas?include_disabled=false&include_hidden=false&limit=100", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schemas: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Schemas []*DetectionSchema `json:"schemas"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Schemas, nil
}
