package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: 3,
		retryDelay: 100 * time.Millisecond,
	}
}

type IngestRequest struct {
	Events []map[string]interface{} `json:"events"`
}

type IngestResponse struct {
	Indexed int      `json:"indexed"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

func (c *Client) StoreEvent(ctx context.Context, event *ocsf.Event) error {
	eventMap := make(map[string]interface{})
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := json.Unmarshal(data, &eventMap); err != nil {
		return fmt.Errorf("convert event to map: %w", err)
	}

	req := IngestRequest{
		Events: []map[string]interface{}{eventMap},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Retry logic for transient failures
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := c.retryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/ingest", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("send request (attempt %d/%d): %w", attempt+1, c.maxRetries+1, err)
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return nil
		}

		// Don't retry on client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			var ingestResp IngestResponse
			if err := json.Unmarshal(respBody, &ingestResp); err == nil && len(ingestResp.Errors) > 0 {
				return fmt.Errorf("storage error: %s", ingestResp.Errors[0])
			}
			return fmt.Errorf("storage returned status %d: %s", resp.StatusCode, string(respBody))
		}

		// Retry on server errors (5xx)
		lastErr = fmt.Errorf("storage returned status %d (attempt %d/%d): %s", resp.StatusCode, attempt+1, c.maxRetries+1, string(respBody))
	}

	return fmt.Errorf("failed after %d attempts: %w", c.maxRetries+1, lastErr)
}
