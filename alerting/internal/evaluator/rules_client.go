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
	url := fmt.Sprintf("%s/schemas?include_disabled=false&include_hidden=false&limit=100", c.baseURL)

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

	// Parse JSON:API response format
	var jsonAPIResponse struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Model      map[string]interface{} `json:"model"`
				View       map[string]interface{} `json:"view"`
				Controller map[string]interface{} `json:"controller"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jsonAPIResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert JSON:API format to DetectionSchema
	schemas := make([]*DetectionSchema, 0, len(jsonAPIResponse.Data))
	for _, item := range jsonAPIResponse.Data {
		schema := &DetectionSchema{
			ID:         item.ID,
			Model:      item.Attributes.Model,
			View:       item.Attributes.View,
			Controller: item.Attributes.Controller,
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}
