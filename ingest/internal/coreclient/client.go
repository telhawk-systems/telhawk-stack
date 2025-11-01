package coreclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
)

// Client communicates with the core normalization service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New constructs a new Client.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// NormalizationResult represents the core response.
type NormalizationResult struct {
	Event json.RawMessage `json:"event"`
}

// Normalize sends the event to core for normalization.
func (c *Client) Normalize(ctx context.Context, event *models.Event) (*NormalizationResult, error) {
	if c == nil {
		return nil, fmt.Errorf("core client not configured")
	}

	payload := event.Raw
	if len(payload) == 0 {
		buf, err := json.Marshal(event.Event)
		if err != nil {
			return nil, fmt.Errorf("serialize event: %w", err)
		}
		payload = buf
	}

	reqBody := map[string]interface{}{
		"id":          event.ID,
		"source":      event.Source,
		"source_type": event.SourceType,
		"format":      "json",
		"payload":     base64.StdEncoding.EncodeToString(payload),
		"received_at": event.Timestamp.UTC().Format(time.RFC3339Nano),
		"attributes": map[string]string{
			"host":         event.Host,
			"hec_token_id": event.HECTokenID,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/normalize", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("core response status %d: %s", resp.StatusCode, errBody["message"])
	}

	var result NormalizationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
