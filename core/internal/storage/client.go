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
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/ingest", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var ingestResp IngestResponse
		if err := json.Unmarshal(respBody, &ingestResp); err == nil && len(ingestResp.Errors) > 0 {
			return fmt.Errorf("storage error: %s", ingestResp.Errors[0])
		}
		return fmt.Errorf("storage returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
